/*
 * Copyright 2019 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// apply applies project configurations to GCP projects.
//
// Usage:
//   $ bazel run :apply -- \
//     --config_path=my_config.yaml
//
// To preview the commands that will run, use `--dry_run`.
package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/apply"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
)

var (
	configPath          = flag.String("config_path", "", "Path to project config file")
	dryRun              = flag.Bool("dry_run", false, "Whether or not to run DPT in the dry run mode. If true, prints the commands that will run without executing.")
	enableTerraform     = flag.Bool("enable_terraform", false, "Whether terraform is preferred over deployment manager.")
	importExisting      = flag.Bool("terraform_import_existing", false, "TERRAFORM ONLY. Whether applicable Terraform resources will try to be imported (used for migrating an existing installation).")
	terraformConfigsDir = flag.String("terraform_configs_dir", "", "TERRAFORM ONLY. Directory path to store generated Terraform configs. The configs are discarded if not specified.")
	projects            arrayFlags
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	// Only allow --projects to be specified once.
	if len(*i) > 0 {
		return errors.New("--projects flag already set")
	}
	for _, v := range strings.Split(value, ",") {
		*i = append(*i, v)
	}
	return nil
}

func (i *arrayFlags) Get() interface{} {
	return i
}

func main() {
	flag.Var(&projects, "projects", "Comma separeted project IDs within --config_path to deploy, or leave unspecified to deploy all projects.")
	flag.Parse()

	if err := applyConfigs(); err != nil {
		log.Fatalf("Failed to apply configs: %v", err)
	}
	log.Println("Setup completed successfully.")
}

// TODO: add tests.
func applyConfigs() (err error) {
	config.EnableTerraform = *enableTerraform
	if *configPath == "" {
		return errors.New("--config_path must be set")
	}

	conf, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	if !*dryRun {
		// Make sure the generated fields file is writable.
		file, err := os.OpenFile(conf.GeneratedFieldsPath, os.O_WRONLY, 0666)
		if err != nil {
			return fmt.Errorf("failed to open %q to write: %v", conf.GeneratedFieldsPath, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("failed to close %q: %v", file.Name(), err)
		}
	}

	// Write generated fields to the output file at the end.
	defer func() {
		if *dryRun {
			// Do nothing.
			return
		}
		if curErr := config.DumpGeneratedFields(conf.AllGeneratedFields, conf.GeneratedFieldsPath); curErr != nil {
			if err == nil {
				err = fmt.Errorf("failed to write generated fields to output file: %v", curErr)
			} else {
				err = fmt.Errorf("%v; failed to write generated fields to output file: %v", err, curErr)
			}
		}
	}()

	if *terraformConfigsDir == "" {
		if *terraformConfigsDir, err = ioutil.TempDir("", ""); err != nil {
			return fmt.Errorf("failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(*terraformConfigsDir)
	}

	*terraformConfigsDir, err = config.NormalizePath(*terraformConfigsDir)
	if err != nil {
		return fmt.Errorf("failed to normalize path %q: %v", *terraformConfigsDir, err)
	}

	// Check if the directory exists and is empty.
	f, err := os.Open(*terraformConfigsDir)
	if err != nil {
		return fmt.Errorf("failed to open %q, if it does not exist, please create it: %v", *terraformConfigsDir, err)
	}
	defer f.Close()

	switch _, err := f.Readdirnames(1); err {
	case nil:
		return fmt.Errorf("%q is not empty", *terraformConfigsDir)
	case io.EOF:
		// Do nothing.
	default:
		return fmt.Errorf("failed to check if directory %q is empty: %v", *terraformConfigsDir, err)
	}

	opts := &apply.Options{DryRun: *dryRun, ImportExisting: *importExisting, TerraformConfigsPath: *terraformConfigsDir}
	var rn runner.Runner
	if *dryRun {
		rn = &runner.Fake{}
	} else {
		rn = &runner.Default{}
	}

	if *enableTerraform {
		return apply.Terraform(conf, projects, opts, rn)
	}

	// DM ONLY CODE.
	// TODO: remove this once DM support is shut down.
	wantProjects := make(map[string]bool)
	for _, p := range projects {
		wantProjects[p] = true
	}

	wantProject := func(project string) bool {
		if len(wantProjects) == 0 {
			return true
		}
		return wantProjects[project]
	}

	// Cannot enable Forseti for remote audit logs project and Forseti project itself until it is deployed.
	enableRemoteAudit := conf.AuditLogsProject != nil && wantProject(conf.AuditLogsProject.ID)

	// Always deploy the remote audit logs project first (if present).
	if enableRemoteAudit {
		log.Printf("Applying config for remote audit log project %q", conf.AuditLogsProject.ID)
		if err := apply.Default(conf, conf.AuditLogsProject, *terraformConfigsDir, rn); err != nil {
			return fmt.Errorf("failed to apply config for remote audit log project %q: %v", conf.AuditLogsProject.ID, err)
		}
	}

	// Deploy the Forseti project.
	if conf.Forseti != nil && wantProject(conf.Forseti.Project.ID) {
		log.Printf("Applying config for Forseti project %q", conf.Forseti.Project.ID)
		// Forseti for Forseti project itself is enabled at the end of apply.Forseti().
		if err := apply.Forseti(conf, *terraformConfigsDir, rn); err != nil {
			return fmt.Errorf("failed to apply config for Forseti project %q: %v", conf.Forseti.Project.ID, err)
		}
	}

	if conf.Forseti != nil && conf.AllGeneratedFields.Forseti.ServiceAccount == "" {
		return fmt.Errorf("forseti project config is specified but has never been deployed")
	}

	// Use conf.AllGeneratedFields.Forseti.ServiceAccount to check if the Forseti project has been deployed or not.
	if enableRemoteAudit && conf.AllGeneratedFields.Forseti.ServiceAccount != "" {
		// Grant Forseti permissions in remote audit log project after Forseti project is deployed.
		workDir, err := terraform.WorkDir(*terraformConfigsDir, conf.AuditLogsProject.ID)
		if err != nil {
			return err
		}
		if err := apply.GrantForsetiPermissions(conf.AuditLogsProject.ID, conf.AllGeneratedFields.Forseti.ServiceAccount, "", workDir, rn); err != nil {
			return fmt.Errorf("failed to grant Forseti permissions to remote audit logs project: %v", err)
		}
	}

	for _, p := range conf.Projects {
		if !wantProject(p.ID) {
			continue
		}
		log.Printf("Applying config for project %q", p.ID)
		if err := apply.Default(conf, p, *terraformConfigsDir, rn); err != nil {
			return fmt.Errorf("failed to apply config for project %q: %v", p.ID, err)
		}
	}

	if conf.AllGeneratedFields.Forseti.ServiceAccount != "" {
		log.Println("Note: Forseti rule generation is no longer run as part of this script. Please use standalone script cmd/rule_generator/rule_generator.go to generate Forseti rules.")
	}

	return nil
}
