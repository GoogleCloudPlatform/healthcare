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

package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

const testCompositeRootResourcesYAML = `
generated_fields_path: generated_fields.yaml

overall:
  organization_id: '12345678'
  folder_id: '98765321'
  billing_account: 000000-000000-000000
  domain: 'my-domain.com'
  allowed_apis:
  - foo-api.googleapis.com
  - bar-api.googleapis.com

forseti:
  project:
    project_id: my-forseti-project
    owners_group: my-forseti-project-owners@my-domain.com
    auditors_group: my-forseti-project-auditors@my-domain.com
    devops:
      state_storage_bucket:
        name: my-forseti-project-state
        location: US
    audit_logs:
      logs_bq_dataset:
        properties:
          name: audit_logs
          location: US
      logs_gcs_bucket:
        ttl_days: 365
        properties:
          name: my-forseti-project-logs
          location: US
          storageClass: MULTI_REGIONAL
    audit:
      logs_bigquery_dataset:
        dataset_id: audit_logs
        location: US
      logs_storage_bucket:
        name: my-forseti-project-logs
        location: US
        storage_class: MULTI_REGIONAL
  properties:
    storage_bucket_location: us-east1
    composite_root_resources: ["folders/111111", "folders/222222"]

projects:
- project_id: my-project
  owners_group: my-project-owners@my-domain.com
  auditors_group: my-project-auditors@my-domain.com
  data_readwrite_groups:
  - my-project-readwrite@my-domain.com
  data_readonly_groups:
  - my-project-readonly@my-domain.com
  - another-readonly-group@googlegroups.com
  devops:
    state_storage_bucket:
      name: my-project-state
      location: US
  enabled_apis:
  - foo-api.googleapis.com
  audit_logs:
    logs_bq_dataset:
      properties:
        name: audit_logs
        location: US
    logs_gcs_bucket:
      ttl_days: 365
      properties:
        name: my-project-logs
        location: US
        storageClass: MULTI_REGIONAL
  audit:
    logs_bigquery_dataset:
      dataset_id: audit_logs
      location: US
    logs_storage_bucket:
      name: my-project-logs
      location: US
      storage_class: MULTI_REGIONAL
{{lpad .ExtraProjectConfig 2}}
`

func TestForseti(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	var got interface{}
	want := map[string]interface{}{
		"project_id": "my-forseti-project",
		"domain":     "my-domain.com",
		"composite_root_resources": []interface{}{
			"organizations/12345678",
			"folders/98765321",
		},
		"storage_bucket_location": "us-east1",
	}
	b, err := yaml.Marshal(conf.Forseti.Properties)
	if err != nil {
		t.Fatalf("yaml.Marshal properties: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}
}

func TestForsetiWithCompositeRootResources(t *testing.T) {
	conf, _ := testconf.ConfigAndProjectWithTestConfigYAML(t, nil, testCompositeRootResourcesYAML)

	var got interface{}
	want := map[string]interface{}{
		"project_id": "my-forseti-project",
		"domain":     "my-domain.com",
		"composite_root_resources": []interface{}{
			"folders/111111",
			"folders/222222",
		},
		"storage_bucket_location": "us-east1",
	}
	b, err := yaml.Marshal(conf.Forseti.Properties)
	if err != nil {
		t.Fatalf("yaml.Marshal properties: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}
}
