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

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

type applyCall struct {
	Config  map[string]interface{}
	Imports []terraform.Import
}

func TestDeployTerraform(t *testing.T) {
	config.EnableTerraform = true
	runner.StubFakeCmds()

	tests := []struct {
		name             string
		data             *testconf.ConfigData
		wantServicesCall *applyCall
		wantUserCall     *applyCall
	}{
		{
			name: "no_resources",
		},
		{
			name: "bigquery_dataset",
			data: &testconf.ConfigData{`
bigquery_datasets:
- dataset_id: foo_dataset
  location: US`},
			wantUserCall: &applyCall{
				Config: unmarshal(t, `
resource:
- google_bigquery_dataset:
    foo_dataset:
      dataset_id: foo_dataset
      project: my-project
      location: US`),
				Imports: []terraform.Import{{
					Address: "google_bigquery_dataset.foo_dataset",
					ID:      "my-project:foo_dataset",
				}},
			},
		},
		{
			name: "compute_image",
			data: &testconf.ConfigData{`
compute_images:
- name: foo-image
  raw_disk:
    source: https://storage.googleapis.com/bosh-cpi-artifacts/bosh-stemcell-3262.4-google-kvm-ubuntu-trusty-go_agent-raw.tar.gz
`},
			wantUserCall: &applyCall{
				Config: unmarshal(t, `
resource:
- google_compute_image:
    foo-image:
      name: foo-image
      project: my-project
      raw_disk:
        source: https://storage.googleapis.com/bosh-cpi-artifacts/bosh-stemcell-3262.4-google-kvm-ubuntu-trusty-go_agent-raw.tar.gz`),
				Imports: []terraform.Import{{
					Address: "google_compute_image.foo-image",
					ID:      "my-project/foo-image",
				}},
			},
		},
		{
			name: "compute_instance",
			data: &testconf.ConfigData{`
compute_instances:
- name: foo-instance
  zone: us-central1-a
  machine_type: n1-standard-1
  boot_disk:
    initialize_params:
      image: debian-cloud/debian-9
  network_interface:
    network: default
`},
			wantUserCall: &applyCall{
				Config: unmarshal(t, `
resource:
- google_compute_instance:
    foo-instance:
      name: foo-instance
      project: my-project
      zone: us-central1-a
      machine_type: n1-standard-1
      boot_disk:
        initialize_params:
          image: debian-cloud/debian-9
      network_interface:
        network: default`),
				Imports: []terraform.Import{{
					Address: "google_compute_instance.foo-instance",
					ID:      "my-project/us-central1-a/foo-instance",
				}},
			},
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
			wantUserCall: &applyCall{
				Config: unmarshal(t, `
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
        'roles/storage.admin user:foo-user@my-domain.com':
          role: roles/storage.admin
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
		{
			name: "project_iam_member",
			data: &testconf.ConfigData{`
project_iam_members:
- role: roles/owner
  member: user:foo-user@my-domain.com`},
			wantUserCall: &applyCall{
				Config: unmarshal(t, `
resource:
- google_project_iam_member:
    project:
      for_each:
        'roles/owner user:foo-user@my-domain.com':
          role: roles/owner
          member: user:foo-user@my-domain.com
      project: my-project
      role: '${each.value.role}'
      member: '${each.value.member}'`),
			},
		},
		{
			name: "project_service",
			data: &testconf.ConfigData{`
project_services:
- service: compute.googleapis.com
- service: iam.googleapis.com`},
			wantServicesCall: &applyCall{
				Config: unmarshal(t, `
terraform:
  required_version: '>= 0.12.0'
  backend:
    gcs:
      bucket: my-project-state
      prefix: services

resource:
- google_project_service:
    project:
      for_each:
        'compute.googleapis.com': true
        'iam.googleapis.com': true
      project: my-project
      service: '${each.key}'`),
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

			if err := Default(conf, project, &Options{DryRun: true, EnableTerraform: true}); err != nil {
				t.Fatalf("deployTerraform: %v", err)
			}

			want := []applyCall{projectCall(t), stateBucketCall(t), auditCall(t)}
			if tc.wantServicesCall != nil {
				want = append(want, *tc.wantServicesCall)
			}
			if tc.wantUserCall != nil {
				addDefaultConfig(t, tc.wantUserCall.Config)
				want = append(want, *tc.wantUserCall)
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("terraform config differs (-got, +want):\n%v", diff)
			}
		})
	}
}

func projectCall(t *testing.T) applyCall {
	return applyCall{
		Config: unmarshal(t, `
terraform:
  required_version: ">= 0.12.0"

resource:
- google_project:
    project:
      project_id: my-project
      name: my-project
      folder_id: '98765321'
      billing_account: 000000-000000-000000`),
		Imports: []terraform.Import{{
			Address: "google_project.project",
			ID:      "my-project",
		}},
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

func auditCall(t *testing.T) applyCall {
	return applyCall{
		Config: unmarshal(t, `
terraform:
  required_version: '>= 0.12.0'
  backend:
    gcs:
      bucket: my-project-state
      prefix: audit-my-project
resource:
- google_logging_project_sink:
    audit-logs-to-bigquery:
      name: audit-logs-to-bigquery
      project: my-project
      filter: 'logName:"logs/cloudaudit.googleapis.com"'
      destination: bigquery.googleapis.com/projects/my-project/datasets/audit_logs
      unique_writer_identity: true
- google_bigquery_dataset:
    audit_logs:
      dataset_id: audit_logs
      project: my-project
      location: US
      access:
      - role: OWNER
        group_by_email: my-project-owners@my-domain.com
      - role: READER
        group_by_email: my-project-auditors@my-domain.com
      - role: WRITER
        user_by_email: '${replace(google_logging_project_sink.audit-logs-to-bigquery.writer_identity, "serviceAccount:", "")}'
- google_storage_bucket:
    my-project-logs:
      name: my-project-logs
      project: my-project
      location: US
      storage_class: MULTI_REGIONAL
      versioning:
        enabled: true
- google_storage_bucket_iam_member:
    my-project-logs:
      for_each:
        'roles/storage.admin group:my-project-owners@my-domain.com':
          role: roles/storage.admin
          member: group:my-project-owners@my-domain.com
        'roles/storage.objectCreator group:cloud-storage-analytics@google.com':
          role: roles/storage.objectCreator
          member: group:cloud-storage-analytics@google.com
        'roles/storage.objectViewer group:my-project-auditors@my-domain.com':
          role: roles/storage.objectViewer
          member: group:my-project-auditors@my-domain.com
      bucket: ${google_storage_bucket.my-project-logs.name}
      role: ${each.value.role}
      member: ${each.value.member}
`),
		Imports: []terraform.Import{
			{Address: "google_logging_project_sink.audit-logs-to-bigquery", ID: "projects/my-project/sinks/audit-logs-to-bigquery"},
			{Address: "google_bigquery_dataset.audit_logs", ID: "my-project:audit_logs"},
			{Address: "google_storage_bucket.my-project-logs", ID: "my-project/my-project-logs"},
		},
	}
}

func addDefaultConfig(t *testing.T, config map[string]interface{}) {
	def := `
terraform:
  required_version: '>= 0.12.0'
  backend:
    gcs:
      bucket: my-project-state
      prefix: user`

	if err := yaml.Unmarshal([]byte(def), &config); err != nil {
		t.Fatalf("json.Unmarshal default config: %v", err)
	}
}

// unmarshal is a helper to unmarshal a yaml or json string to an interface (map).
// Note: the actual configs will always be json, but we allow yaml in tests to make them easier to write in test cases.
func unmarshal(t *testing.T, s string) map[string]interface{} {
	out := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}
	return out
}
