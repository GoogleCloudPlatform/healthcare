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
)

// The following vars are stubbed in tests.
var cmdRun = func(cmd *exec.Cmd) error {
	log.Printf("Running: %v", cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Apply applies the config. The config will be written as a .tf.json file in the given dir.
func Apply(config *Config, dir string) error {
	runCmd := func(args ...string) error {
		cmd := exec.Command("terraform", args...)
		cmd.Dir = dir
		return cmdRun(cmd)
	}
	b, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal terraform config: %v", err)
	}

	log.Printf("forseti terraform config:\n%v", string(b))

	// drw-r--r--
	if err := ioutil.WriteFile(filepath.Join(dir, "main.tf.json"), b, 0644); err != nil {
		return fmt.Errorf("failed to write terraform config: %v", err)
	}

	if err := runCmd("init"); err != nil {
		return fmt.Errorf("failed to init terraform dir: %v", err)
	}
	if err := runCmd("apply"); err != nil {
		return fmt.Errorf("failed to apply plan: %v", err)
	}
	return nil

}
