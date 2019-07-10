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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
	tests := []struct {
		name      string
		inputPath string
		wantPath  string
	}{
		{
			name:      "spanned_configs",
			inputPath: "deploy/samples/spanned_configs/root.yaml",
			wantPath:  "deploy/samples/project_with_remote_audit_logs.yaml",
		},
		{
			name:      "template",
			inputPath: "deploy/samples/template/input.yaml",
			wantPath:  "deploy/samples/minimal.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := config.Load(tc.inputPath)
			if err != nil {
				t.Fatalf("config.Load = %v", err)
			}
			want, err := config.Load(tc.wantPath)
			if err != nil {
				t.Fatalf("config.Load = %v", err)
			}
			allowUnexported := cmp.AllowUnexported(
				config.BigqueryDataset{}, config.DefaultResource{}, config.ForsetiProperties{},
				config.GCSBucket{}, config.LifecycleRule{}, config.IAMPolicy{}, config.Metric{},
				config.Pubsub{}, config.Subscription{},
			)
			opts := []cmp.Option{
				allowUnexported,
				cmpopts.SortSlices(func(a, b *config.Project) bool { return a.ID < b.ID }),
			}
			if diff := cmp.Diff(got, want, opts...); diff != "" {
				t.Fatalf("yaml differs (-got +want):\n%v", diff)
			}
		})
	}
}

func TestPattern(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("ioutil.TempDir = %v", err)
	}
	defer os.RemoveAll(dir)

	afn := filepath.Join(dir, "a.yaml")
	ac := []byte("imports: [{pattern: '*.yaml'}]")
	if err := ioutil.WriteFile(afn, ac, 0664); err != nil {
		t.Fatalf("ioutil.WriteFile = %v", err)
	}

	bfn := filepath.Join(dir, "b.yaml")
	bc := []byte(`
overall:
 organization_id: '123'
 domain: foo.com
`)
	if err := ioutil.WriteFile(bfn, bc, 0664); err != nil {
		t.Fatalf("ioutil.WriteFile = %v", err)
	}
	got, err := config.Load(afn)
	if err != nil {
		t.Fatalf("config.Load = %v", err)
	}
	want, err := config.Load(bfn)
	if err != nil {
		t.Fatalf("config.Load = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("config differs (-got +want):\n%v", diff)
	}
}
