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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/imdario/mergo"
)

// ActionFunc are functions that can implement extra actions to run on existing Terraform deployments.
type ActionFunc func(dir string, rn runner.Runner) error

// Options configure a terraform apply call.
type Options struct {
	Imports      []Import
	CustomConfig map[string]interface{}
	ApplyFlags   []string
	ExtraActions []ActionFunc
}

// Apply applies the config. The config will be written as a .tf.json file in the given dir.
// All imports in opts.Imports will be imported prior being applied.
// Thus, if a resource exists it will be imported to the terraform state.
// Without importing an existing resource terraform can fail with an "ALREADY EXISTS" error when it tries to create it.
func Apply(config *Config, dir string, opts *Options, rn runner.Runner) error {
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
		if err := rn.CmdRun(exec.Command("cp", "-r", "-L", "--no-preserve=mode,ownership", m.Source, dst)); err != nil {
			return fmt.Errorf("failed to copy %q to %q: %v", m.Source, dst, err)
		}
	}

	runCmd := func(args ...string) error {
		cmd := exec.Command("terraform", args...)
		cmd.Dir = dir
		return rn.CmdRun(cmd)
	}
	b, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal terraform config: %v", err)
	}

	if len(opts.CustomConfig) > 0 {
		orig := make(map[string]interface{})
		if err := json.Unmarshal(b, &orig); err != nil {
			return fmt.Errorf("failed to marshal config to map: %v", err)
		}
		if err := mergo.Merge(&orig, opts.CustomConfig, mergo.WithAppendSlice); err != nil {
			return fmt.Errorf("failed to merge original config with custom: %v", err)
		}
		b, err = json.MarshalIndent(orig, "", " ")
		if err != nil {
			return fmt.Errorf("failed to marshal merged config: %v", err)
		}
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
		if err := runCmd("import", "-no-color", imp.Address, imp.ID); err != nil {
			log.Print(err)
		}
	}

	if len(opts.ExtraActions) > 0 {
		exists, err := deploymentExists(dir, rn)
		if err != nil {
			return err
		}
		if exists {
			for _, ea := range opts.ExtraActions {
				if err := ea(dir, rn); err != nil {
					return err
				}
			}
		}

	}

	if err := runCmd(append([]string{"apply"}, opts.ApplyFlags...)...); err != nil {
		return fmt.Errorf("failed to apply plan: %v", err)
	}
	return nil
}

// WorkDir creates and returns the directory path to run the current terraform commands.
func WorkDir(base string, subdirs ...string) (string, error) {
	if base == "" {
		return "", errors.New("base directory path must not be empty")
	}
	dir := filepath.Join(append([]string{base}, subdirs...)...)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to mkdir %q: %v", dir, err)
	}
	return dir, nil
}

func deploymentExists(dir string, rn runner.Runner) (bool, error) {
	// Only run extra actions if the deployment already exists.
	cmd := exec.Command("terraform", "show", "-json")
	cmd.Dir = dir
	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return false, err
	}

	type showResult struct {
		Values *interface{} `json:"values"`
	}
	sr := new(showResult)
	if err := json.Unmarshal(out, sr); err != nil {
		return false, err
	}
	return sr.Values != nil, nil
}
