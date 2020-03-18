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

// Policy Generator automate generation of Google-recommended Policy Library constraints based on your Terraform configs.
//
// Usage:
// $ bazel run :policygen -- --config_path=/path/to/config --output_dir=/tmp/constraints
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"flag"
	
	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/policygen/terraform"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

var (
	// TODO: make it work with Terraform Engine configs.
	configPath = flag.String("config_path", "", "Path to Terraform config")
	outputDir  = flag.String("output_dir", "", "Path to directory to generate constraints")
)

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("--config_path must be set")
	}
	if *outputDir == "" {
		log.Fatal("--output_dir must be set")
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {

	var err error
	*configPath, err = config.NormalizePath(*configPath)
	if err != nil {
		return fmt.Errorf("normalize path %q: %v", *configPath, err)
	}

	*outputDir, err = config.NormalizePath(*outputDir)
	if err != nil {
		return fmt.Errorf("normalize path %q: %v", *outputDir, err)
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	rn := &runner.Default{}
	tfCmd := func(args ...string) error {
		cmd := exec.Command("terraform", args...)
		cmd.Dir = tmpDir
		return rn.CmdRun(cmd)
	}
	tfCmdOutput := func(args ...string) ([]byte, error) {
		cmd := exec.Command("terraform", args...)
		cmd.Dir = tmpDir
		return rn.CmdOutput(cmd)
	}

	cp := exec.Command("cp", *configPath, tmpDir)
	if err := rn.CmdRun(cp); err != nil {
		return fmt.Errorf("copy configs to temp directory: %v", err)
	}

	if err := tfCmd("init"); err != nil {
		return fmt.Errorf("terraform init: %v", err)
	}

	planPath := filepath.Join(tmpDir, "plan.tfplan")
	if err := tfCmd("plan", "-out", planPath); err != nil {
		return fmt.Errorf("terraform plan: %v", err)
	}

	b, err := tfCmdOutput("show", "-json", planPath)
	if err != nil {
		return fmt.Errorf("terraform show: %v", err)
	}

	resources, err := terraform.ReadPlanResources(b)
	if err != nil {
		return fmt.Errorf("unmarshal json: %v", err)
	}

	for _, r := range resources {
		log.Printf("%+v", r)
	}

	// TODO: generate policies that do not rely on input config.
	// TODO: generate policies that rely on input config.

	return nil
}
