package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

// TODO: add a test case for remote audit project once test configs
// can be generated with a remote audit project.
func TestResourceRules(t *testing.T) {
	configData := &ConfigData{`
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

	conf, _ := getTestConfigAndProject(t, configData)
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
