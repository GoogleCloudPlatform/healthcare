// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apply

import (
	"encoding/json"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

type applyCall struct {
	Config  interface{}
	Imports []terraform.Import
}

func TestDeployTerraform(t *testing.T) {
	tests := []struct {
		name string
		data *testconf.ConfigData
		want *applyCall
	}{
		{
			name: "no_resources",
		},
		{
			name: "storage_bucket",
			data: &testconf.ConfigData{`
storage_buckets:
- name: foo-bucket
  location: US
  _iam_members:
  - role: roles/storage.admin
    member: user:foo-user@my-domain.com`},
			want: &applyCall{
				Config: unmarshal(t, `
terraform:
  required_version: ">= 0.12.0"
  backend:
    gcs:
      bucket: my-project-state
      prefix: resources

resource:
- google_storage_bucket:
    foo-bucket:
      name: foo-bucket
      project: my-project
      location: US
      versioning:
        enabled: true
- google_storage_bucket_iam_member:
    foo-bucket:
      for_each:
      - role: roles/storage.admin
        member: user:foo-user@my-domain.com
      bucket: '${google_storage_bucket.foo-bucket.name}'
      role: '${each.value.role}'
      member: '${each.value.member}'`),
				Imports: []terraform.Import{{
					Address: "google_storage_bucket.foo-bucket",
					ID:      "my-project/foo-bucket",
				}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conf, project := testconf.ConfigAndProject(t, tc.data)

			var got []applyCall
			terraformApply = func(config *terraform.Config, _ string, opts *terraform.Options) error {
				b, err := json.Marshal(config)
				if err != nil {
					t.Fatalf("json.Marshal(%s) = %v", config, err)
				}

				call := applyCall{
					Config: unmarshal(t, string(b)),
				}
				if opts != nil {
					call.Imports = opts.Imports
				}
				got = append(got, call)
				return nil
			}

			if err := defaultTerraform(conf, project); err != nil {
				t.Fatalf("deployTerraform: %v", err)
			}

			want := []applyCall{stateBucketCall(t)}
			if tc.want != nil {
				want = append(want, *tc.want)
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("terraform config differs (-got, +want):\n%v", diff)
			}
		})
	}
}

func stateBucketCall(t *testing.T) applyCall {
	return applyCall{
		Config: unmarshal(t, `
terraform:
  required_version: ">= 0.12.0"

resource:
- google_storage_bucket:
    my-project-state:
      name: my-project-state
      project: my-project
      location: US
      versioning:
        enabled: true`),
		Imports: []terraform.Import{{
			Address: "google_storage_bucket.my-project-state",
			ID:      "my-project/my-project-state",
		}},
	}
}

// unmarshal is a helper to unmarshal a yaml or json string to an interface (map).
// Note: the actual configs will always be json, but we allow yaml in tests to make them easier to write in test cases.
func unmarshal(t *testing.T, s string) interface{} {
	var out interface{}
	if err := yaml.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("json.Unmarshal = %v", err)
	}
	return out
}
