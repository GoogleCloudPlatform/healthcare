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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestNormalizePath(t *testing.T) {
	relativePath := "samples/project_with_remote_audit_logs.yaml"
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
generated_fields_path: bar.yaml
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
generated_fields_path: bar.yaml
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
		name                string
		inputPath           string
		wantPath            string
		wantGenfieldsSuffix string
	}{
		{
			name:                "spanned_configs",
			inputPath:           "samples/spanned_configs/root.yaml",
			wantPath:            "samples/project_with_remote_audit_logs.yaml",
			wantGenfieldsSuffix: "samples/spanned_configs/generated_fields.yaml",
		},
		{
			name:                "template",
			inputPath:           "samples/template/input.yaml",
			wantPath:            "samples/minimal.yaml",
			wantGenfieldsSuffix: "samples/generated_fields.yaml",
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
				config.Pubsub{}, config.Subscription{}, tfconfig.StorageBucket{},
			)
			opts := []cmp.Option{
				allowUnexported,
				cmpopts.SortSlices(func(a, b *config.Project) bool { return a.ID < b.ID }),
			}
			if !strings.Contains(got.GeneratedFieldsPath, tc.wantGenfieldsSuffix) {
				t.Fatalf("generated fields path %q does not contain suffix %q", got.GeneratedFieldsPath, tc.wantGenfieldsSuffix)
			}
			if diff := cmp.Diff(got, want, opts...); diff != "" {
				t.Fatalf("yaml differs (-got +want):\n%v", diff)
			}
		})
	}
}

func TestLoadGeneratedFields(t *testing.T) {
	tests := []struct {
		name      string
		inputConf []byte
		wantErr   bool
	}{
		{
			name: "valid_genfields_path",
			inputConf: []byte(`
overall:
  billing_account: 000000-000000-000000
generated_fields_path: a/b/c/generated_fields.yaml
projects: []
`),
			wantErr: false,
		},
		{
			name: "invalid_empty_genfields_path",
			inputConf: []byte(`
overall:
  billing_account: 000000-000000-000000
projects: []
`),
			wantErr: true,
		},
		{
			name: "invalid_absolute_genfields_path",
			inputConf: []byte(`
overall:
  billing_account: 000000-000000-000000
generated_fields_path: /a/b/c/generated_fields.yaml
projects: []
`),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conf, err := ioutil.TempFile("", "")
			if err != nil {
				t.Fatalf("ioutil.TempFile: %v", err)
			}
			defer os.Remove(conf.Name())
			if _, err := conf.Write(tc.inputConf); err != nil {
				t.Fatalf("os.File.Write: %v", err)
			}
			if _, err := config.Load(conf.Name()); (err != nil) != tc.wantErr {
				t.Fatalf("config.Load = error: %v; want error %t", err, tc.wantErr)
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
generated_fields_path: bar.yaml
overall:
  billing_account: 000000-000000-000000
  organization_id: '12345678'
  domain: foo.com
projects: []
`)
	if err := ioutil.WriteFile(bfn, bc, 0664); err != nil {
		t.Fatalf("ioutil.WriteFile = %v", err)
	}
	got, err := config.Load(afn)
	if err != nil {
		t.Fatalf("config.Load a.yaml = %v", err)
	}
	want, err := config.Load(bfn)
	if err != nil {
		t.Fatalf("config.Load b.yaml = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("config differs (-got +want):\n%v", diff)
	}
}
