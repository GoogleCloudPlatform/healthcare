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

var locConfigData = &testconf.ConfigData{`
resources:
  bq_datasets:
  - properties:
      name: foo-dataset
      location: US
  gce_instances:
  - properties:
      name: foo-instance
      zone: us-central1-f
      diskImage: projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts
      machineType: f1-micro
  gcs_buckets:
  - properties:
      name: my-project-foo-bucket
      location: us-central1`}

const wantLocationYAML = `
- name: Global location whitelist.
  mode: whitelist
  resource:
  - type: organization
    resource_ids:
    - '12345678'
  applies_to:
  - type: '*'
    resource_ids:
    - '*'
  locations:
  - US
  - US-CENTRAL1
  - US-CENTRAL1-F
- name: Project my-forseti-project audit logs bucket location whitelist.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-forseti-project
  applies_to:
  - type: bucket
    resource_ids:
    - my-forseti-project-logs
  locations:
  - US
- name: Project my-forseti-project audit logs dataset location whitelist.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-forseti-project
  applies_to:
  - type: dataset
    resource_ids:
    - my-forseti-project:audit_logs
  locations:
  - US
- name: Project my-project resource whitelist for location US.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-project
  applies_to:
  - type: dataset
    resource_ids:
    - my-project:foo-dataset
  locations:
  - US
- name: Project my-project resource whitelist for location US-CENTRAL1.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-project
  applies_to:
  - type: bucket
    resource_ids:
    - my-project-foo-bucket
  locations:
    - US-CENTRAL1
- name: Project my-project resource whitelist for location US-CENTRAL1-F.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-project
  applies_to:
  - type: instance
    resource_ids:
    - '123'
  locations:
  - US-CENTRAL1-F
- name: Project my-project audit logs bucket location whitelist.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-project
  applies_to:
  - type: bucket
    resource_ids:
    - my-project-logs
  locations:
  - US
- name: Project my-project audit logs dataset location whitelist.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-project
  applies_to:
  - type: dataset
    resource_ids:
    - my-project:audit_logs
  locations:
  - US
`

func TestLocationRules(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, locConfigData)
	got, err := LocationRules(conf)
	if err != nil {
		t.Fatalf("LocationRules = %v", err)
	}

	want := make([]LocationRule, 0)
	if err := yaml.Unmarshal([]byte(wantLocationYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
