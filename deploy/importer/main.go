/*
 * Copyright 2020 Google LLC.
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

// The Terraform Importer imports resources from existing infrastructure into Terraform configs.
// Requires Terraform to be installed and for authentication to be configured for each provider in the Terraform configs provider blocks.
//
// Usage:
// $ bazel run :importer -- --input_dir=/path/to/config
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/importer/tfimport"
	"github.com/GoogleCloudPlatform/healthcare/deploy/policygen/terraform"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

var (
	inputDir = flag.String("input_dir", ".", "Required. Path to Terraform config.")
)

func main() {
	flag.Parse()

	if *inputDir == "" {
		log.Fatalf("--input_dir must be set and not be empty")
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Normalize the config path.
	var err error
	*inputDir, err = config.NormalizePath(*inputDir)
	if err != nil {
		return fmt.Errorf("normalize path %q: %v", *inputDir, err)
	}

	// Create Terraform command runners.
	rn := &runner.Default{}
	tfCmd := func(args ...string) error {
		cmd := exec.Command("terraform", args...)
		cmd.Dir = *inputDir
		return rn.CmdRun(cmd)
	}
	tfCmdOutput := func(args ...string) ([]byte, error) {
		cmd := exec.Command("terraform", args...)
		cmd.Dir = *inputDir
		return rn.CmdOutput(cmd)
	}

	// Init is safe to run on an already-initialized config dir.
	if err := tfCmd("init"); err != nil {
		return fmt.Errorf("terraform init: %v", err)
	}

	// Generate and load the plan using a temp var.
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	planPath := tmpfile.Name()
	if err := tfCmd("plan", "-out", planPath); err != nil {
		return fmt.Errorf("terraform plan: %v", err)
	}
	b, err := tfCmdOutput("show", "-json", planPath)
	if err != nil {
		return fmt.Errorf("terraform show: %v", err)
	}

	// Load only "create" changes.
	createChanges, err := terraform.ReadPlanChanges(b, []string{"create"})
	if err != nil {
		return fmt.Errorf("read Terraform plan changes: %q", err)
	}

	// Import all importable create changes.
	importedSomething := false
	for _, cc := range createChanges {
		// Get the provider config values (pcv) for this particular resource.
		// This is needed to determine if it's possible to import the resource.
		pcv, err := terraform.ReadProviderConfigValues(b, cc.Kind, cc.Name)
		if err != nil {
			return fmt.Errorf("read provider config values from the Terraform plan: %q", err)
		}

		// Try to convert to an importable resource.
		ir, ok := tfimport.Importable(cc, pcv)
		if !ok {
			log.Printf("Resource %q of type %q not importable\n", cc.Address, cc.Kind)
			continue
		}
		log.Printf("Found importable resource: %q\n", ir.Change.Address)

		// Attempt the import.
		output, err := tfimport.Import(rn, ir, *inputDir)

		// Handle the different outcomes of the import attempt.
		switch {
		// err will only be nil when the import succeed.
		case err == nil:
			// Import succeeded, print the success output.
			fmt.Println(string(output))
			importedSomething = true

		// err will be `exit code 1` even when it failed because the resource is not importable or already exists.
		case tfimport.NotImportable(output):
			log.Printf("Import not supported by provider for resource %q\n", ir.Change.Address)
		case tfimport.DoesNotExist(output):
			log.Printf("Resource %q does not exist, not importing\n", ir.Change.Address)

		// Important to handle this last.
		default:
			return fmt.Errorf("import resource %q: %v", ir.Change.Address, err)
		}
	}

	if !importedSomething {
		log.Printf("No resources imported.")
	}

	return nil
}
