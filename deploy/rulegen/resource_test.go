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

package rulegen

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

// TODO: add a test case for remote audit project once test configs
// can be generated with a remote audit project.
func TestResourceRules(t *testing.T) {
	configData := &testconf.ConfigData{`
resources:
 bq_datasets:
 - properties:
      name: foo-dataset
      location: US
 gce_instances:
 - properties:
      name: foo-instance
      zone: us-east1-a
      diskImage: projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts
      machineType: f1-micro
 gcs_buckets:
 - properties:
      name: foo-bucket
      location: us-east1
`}
	wantYAML := `
- name: 'Project resource trees.'
  mode: required
  resource_types:
  - project
  - bucket
  - dataset
  - instance
  resource_trees:
  - type: project
    resource_id: '*'
  - type: project
    resource_id: my-forseti-project
    children:
    - type: dataset
      resource_id: my-forseti-project:audit_logs
    - type: bucket
      resource_id: my-forseti-project-logs
  - type: project
    resource_id: my-project
    children:
    - type: dataset
      resource_id: my-project:audit_logs
    - type: bucket
      resource_id: my-project-logs
    - type: bucket
      resource_id: foo-bucket
    - type: dataset
      resource_id: my-project:foo-dataset
    - type: instance
      resource_id: '123'
`

	conf, _ := testconf.ConfigAndProject(t, configData)
	got, err := ResourceRules(conf)
	if err != nil {
		t.Fatalf("ResourceRules = %v", err)
	}

	want := make([]ResourceRule, 0)
	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
