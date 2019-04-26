package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

var locConfigData = &ConfigData{`
resources:
- bigquery_dataset:
    properties:
      name: foo-dataset
      location: US
- gce_instance:
    properties:
      name: foo-instance
      zone: us-central1-f
      diskImage: projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts
      machineType: f1-micro
- gcs_bucket:
    properties:
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
	config, _ := getTestConfigAndProject(t, locConfigData)
	got, err := LocationRules(config)
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
