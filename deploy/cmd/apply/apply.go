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
//     --config_path=my_config.yaml \
//     --output_path=my_output.yaml \
//
// To preview the commands that will run, use `--dry_run`.
package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/apply"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/rulegen"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

var (
	configPath      = flag.String("config_path", "", "Path to project config file")
	outputPath      = flag.String("output_path", "", "Path to output file to write generated fields")
	rulesPath       = flag.String("rules_path", "", "Path to local directory or GCS bucket to output rules files. If unset, directly writes to the Forseti server bucket.")
	dryRun          = flag.Bool("dry_run", false, "Whether or not to run DPT in the dry run mode. If true, prints the commands that will run without executing.")
	enableTerraform = flag.Bool("enable_terraform", false, "DEV ONLY. Whether terraform is preferred over deployment manager.")
	projects        arrayFlags
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

	if *configPath == "" {
		log.Fatal("--config_path must be set")
	}
	if *outputPath == "" {
		log.Fatal("--output_path must be set")
	}
	if *dryRun {
		runner.StubFakeCmds()
	}
	if err := run(); err != nil {
		log.Fatalf("Setup failed: %v", err)
	}
	log.Println("Setup completed successfully.")
}

func run() (err error) {
	conf, err := config.Load(*configPath, *outputPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

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

	// Write generated fields to the output file at the end.
	defer func() {
		if *dryRun {
			// Do nothing.
			return
		}
		if curErr := config.DumpGeneratedFields(conf.AllGeneratedFields, *outputPath); curErr != nil {
			if err == nil {
				err = fmt.Errorf("failed to write generated fields to output file: %v", curErr)
			} else {
				err = fmt.Errorf("%v; failed to write generated fields to output file: %v", err, curErr)
			}
		}
	}()

	enableForseti := conf.Forseti != nil && wantProject(conf.Forseti.Project.ID)
	enableRemoteAudit := conf.AuditLogsProject != nil && wantProject(conf.AuditLogsProject.ID)

	// Cannot enable Forseti for remote audit logs project and Forseti project itself until it is deployed.
	opts := &apply.Options{EnableTerraform: *enableTerraform, DryRun: *dryRun}

	// Always deploy the remote audit logs project first (if present).
	if enableRemoteAudit {
		log.Printf("Applying config for remote audit log project %q", conf.AuditLogsProject.ID)
		if err := apply.Default(conf, conf.AuditLogsProject, opts); err != nil {
			return fmt.Errorf("failed to apply config for remote audit log project %q: %v", conf.AuditLogsProject.ID, err)
		}
	}
	if enableForseti {
		log.Printf("Applying config for Forseti project %q", conf.Forseti.Project.ID)
		// Forseti for Forseti project itself is enabled at the end of apply.Forseti().
		if err := apply.Forseti(conf, opts); err != nil {
			return fmt.Errorf("failed to apply config for Forseti project %q: %v", conf.Forseti.Project.ID, err)
		}
		if enableRemoteAudit {
			// Grant Forseti permissions in remote audit log project after Forseti project is deployed.
			if err := apply.GrantForsetiPermissions(conf.AuditLogsProject.ID, conf.AllGeneratedFields.Forseti.ServiceAccount); err != nil {
				return fmt.Errorf("failed to grant Forseti permissions to remote audit logs project: %v", err)
			}
		}
	}
	opts.EnableForseti = enableForseti
	for _, p := range conf.Projects {
		if !wantProject(p.ID) {
			continue
		}
		log.Printf("Applying config for project %q", p.ID)
		if err := apply.Default(conf, p, opts); err != nil {
			return fmt.Errorf("failed to apply config for project %q: %v", p.ID, err)
		}
	}

	log.Println("Running rule generator.")
	if err := rulegen.Run(conf, *rulesPath); err != nil {
		return fmt.Errorf("failed to generate Forseti rules: %v", err)
	}
	log.Println("Rule generation successful")

	return nil
}
