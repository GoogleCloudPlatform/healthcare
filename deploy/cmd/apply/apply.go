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
	"io/ioutil"
	"log"
	"os"
	"strings"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/apply"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/google/shlex"
)

var (
	configPath      = flag.String("config_path", "", "Path to project config file")
	dryRun          = flag.Bool("dry_run", false, "Whether or not to run DPT in the dry run mode. If true, prints the commands that will run without executing.")
	enableTerraform = flag.Bool("enable_terraform", true, "This flag cannot be false. Deployment manager is no longer supported.")

	importExisting = flag.Bool("terraform_import_existing", false, "TERRAFORM ONLY. Whether applicable Terraform resources will try to be imported (used for migrating an existing installation).")

	terraformConfigsDir = flag.String("terraform_configs_dir", "", "TERRAFORM ONLY. Directory path to store generated Terraform configs. The configs are discarded if not specified.")
	terraformApplyFlags = flag.String("terraform_apply_flags", "", "TERRAFORM ONLY. Extra option flags to pass to apply command.")
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

	if !*enableTerraform {
		log.Fatal("--enable_terraform must be unset or true. Deployment manager is no longer supported.")
	}

	if err := applyConfigs(); err != nil {
		log.Fatalf("Failed to apply configs: %v", err)
	}
	log.Println("Setup completed successfully.")
}

// TODO: add tests.
func applyConfigs() (err error) {
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

	if err := os.MkdirAll(*terraformConfigsDir, 0755); err != nil {
		return fmt.Errorf("failed to mkdir %q: %v", *terraformConfigsDir, err)
	}

	opts := &apply.Options{DryRun: *dryRun, ImportExisting: *importExisting, TerraformConfigsPath: *terraformConfigsDir}
	if *terraformApplyFlags != "" {
		var err error
		opts.TerraformApplyFlags, err = shlex.Split(*terraformApplyFlags)
		if err != nil {
			return fmt.Errorf("failed to shell split terraform apply options: %v", err)
		}
	}

	var rn runner.Runner
	if *dryRun {
		rn = &runner.Fake{}
	} else {
		rn = &runner.Default{}
	}

	return apply.Terraform(conf, projects, opts, rn)
}
