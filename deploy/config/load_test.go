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
	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
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

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		inputConf []byte
		ok        bool
	}{
		{
			name: "valid_config",
			inputConf: []byte(`
overall:
  billing_account: 000000-000000-000000
  organization_id: '12345678'
  domain: foo.com
projects: []
`),
			ok: true,
		},
		{
			name: "invalid_config",
			inputConf: []byte(`
overall:
  billing_account: 000000-000000-000000
  organization_id: '12345678'
  domain: foo.com
`),
			ok: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := config.ValidateConf(tc.inputConf); (err == nil) != tc.ok {
				t.Fatalf("config.Validate = %t, want %t", err == nil, tc.ok)
			}
		})
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

	// Just make sure generated fields are parsed correctly.
	genFieldsFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("ioutil.TempFile: %v", err)
	}
	defer func() {
		os.Remove(genFieldsFile.Name())
	}()
	genFieldsFile.Write([]byte(`
projects:
  foo-project:
    log_sink_service_account: some-sa@gcp-sa-logging.iam.gserviceaccount.com
    project_number: '123'`))

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := config.Load(tc.inputPath, genFieldsFile.Name())
			if err != nil {
				t.Fatalf("config.Load = %v", err)
			}
			want, err := config.Load(tc.wantPath, genFieldsFile.Name())
			if err != nil {
				t.Fatalf("config.Load = %v", err)
			}
			allowUnexported := cmp.AllowUnexported(
				config.BigqueryDataset{}, config.DefaultResource{}, config.ForsetiProperties{},
				config.GCSBucket{}, config.LifecycleRule{}, config.IAMPolicy{}, config.Metric{},
				config.Pubsub{}, config.Subscription{}, tfconfig.StorageBucket{},
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
  billing_account: 000000-000000-000000
  organization_id: '12345678'
  domain: foo.com
projects: []
`)
	if err := ioutil.WriteFile(bfn, bc, 0664); err != nil {
		t.Fatalf("ioutil.WriteFile = %v", err)
	}

	// use different file extension so pattern doesn't parse it
	gfn := filepath.Join(dir, "generated_fields.txt")
	if _, err := os.Create(gfn); err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	got, err := config.Load(afn, gfn)
	if err != nil {
		t.Fatalf("config.Load a.yaml = %v", err)
	}
	want, err := config.Load(bfn, gfn)
	if err != nil {
		t.Fatalf("config.Load b.yaml = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("config differs (-got +want):\n%v", diff)
	}
}
