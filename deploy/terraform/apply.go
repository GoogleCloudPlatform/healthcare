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

package terraform

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

// Options configure a terraform apply call.
type Options struct {
	Imports []Import
}

// Apply applies the config. The config will be written as a .tf.json file in the given dir.
func Apply(config *Config, dir string, opts *Options) error {
	if opts == nil {
		opts = new(Options)
	}

	// Copy modules to the running dir from Bazel cache.
	// Terraform needs write access to the modules which Bazel's cache does not allow.
	dstMap := make(map[string]bool)
	for _, m := range config.Modules {
		dst := filepath.Join(dir, filepath.Dir(m.Source))
		if dstMap[dst] {
			continue
		}
		dstMap[dst] = true
		if err := os.MkdirAll(dst, os.ModePerm); err != nil {
			return fmt.Errorf("failed to mkdir %q: %v", dst, err)
		}
		if err := runner.CmdRun(exec.Command("cp", "-r", m.Source, dst)); err != nil {
			return fmt.Errorf("failed to copy %q to %q: %v", m.Source, dst, err)
		}
	}

	runCmd := func(args ...string) error {
		cmd := exec.Command("terraform", args...)
		cmd.Dir = dir
		return runner.CmdRun(cmd)
	}
	b, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal terraform config: %v", err)
	}

	log.Printf("terraform config:\n%v", string(b))

	// drw-r--r--
	if err := ioutil.WriteFile(filepath.Join(dir, "main.tf.json"), b, 0644); err != nil {
		return fmt.Errorf("failed to write terraform config: %v", err)
	}

	if err := runCmd("init"); err != nil {
		return fmt.Errorf("failed to init terraform dir: %v", err)
	}
	for _, imp := range opts.Imports {
		// TODO: this will fail if the resource does not exist
		// or is already a part of the state. Avoid this in the long run.
		// For the time being, ignore the error and just log it.
		if err := runCmd("import", imp.Address, imp.ID); err != nil {
			log.Print(err)
		}
	}

	if err := runCmd("apply"); err != nil {
		return fmt.Errorf("failed to apply plan: %v", err)
	}
	return nil
}
