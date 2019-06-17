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

package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

// NormalizePath normalizes paths specified through a local run or Bazel invocation.
func NormalizePath(path string) (string, error) {
	path, err := homedir.Expand(path)
	if err != nil {
		return "", err
	}
	path = os.ExpandEnv(path)
	if strings.HasPrefix(path, "gs://") || filepath.IsAbs(path) {
		return path, nil
	}
	// Path is relative from where the script was launched from.
	// When using `bazel run`, the environment variable BUILD_WORKING_DIRECTORY
	// will be set to the path where the command was run from.
	cwd := os.Getenv("BUILD_WORKING_DIRECTORY")
	if len(cwd) == 0 {
		if cwd, err = os.Getwd(); err != nil {
			return "", err
		}
	}
	return filepath.Abs(filepath.Join(cwd, path))
}
