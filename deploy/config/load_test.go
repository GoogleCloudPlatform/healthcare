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

package config_test

import (
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
)

func TestNormalizePath(t *testing.T) {
	relativePath := "deploy/samples/project_with_remote_audit_logs.yaml"
	path, err := config.NormalizePath(relativePath)
	if err != nil {
		t.Fatalf("cannot normalizePath: %q", relativePath)
	}
	if _, err = os.Stat(path); err != nil {
		t.Fatalf("cannot find project_with_remote_audit_logs.yaml: %q", path)
	}
}

func TestLoad(t *testing.T) {
	confPath, err := config.NormalizePath("deploy/testconf/test_multiple_yaml/conf.yaml")
	if err != nil {
		t.Fatalf("NormalizePath error: %v", err)
	}
	conf, err := config.Load(confPath)
	if err != nil {
		t.Fatalf("config.Load = %v", err)
	}
	expectedPath, err := config.NormalizePath("deploy/testconf/test_multiple_yaml/expected.yaml")
	if err != nil {
		t.Fatalf("NormalizePath error: %v", err)
	}
	expectedConf, err := config.Load(expectedPath)
	if err != nil {
		t.Fatalf("config.Load = %v", err)
	}

	opt := cmp.AllowUnexported(config.BigqueryDataset{}, config.DefaultResource{}, config.Metric{})
	if diff := cmp.Diff(conf.Projects, expectedConf.Projects, opt); diff != "" {
		t.Fatalf("yaml differs (-got +want):\n%v", diff)
	}
}
