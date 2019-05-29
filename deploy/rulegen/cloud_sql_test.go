package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestCloudSQLRules(t *testing.T) {
	conf, _ := getTestConfigAndProject(t, nil)
	got, err := CloudSQLRules(conf)
	if err != nil {
		t.Fatalf("CloudSQLRules = %v", err)
	}

	wantYAML := `
- name: Disallow publicly exposed cloudsql instances (SSL disabled).
  resource:
  - type: organization
    resource_ids:
    - '12345678'
  instance_name: '*'
  authorized_networks: 0.0.0.0/0
  ssl_enabled: 'false'
- name: Disallow publicly exposed cloudsql instances (SSL enabled).
  resource:
  - type: organization
    resource_ids:
    - '12345678'
  instance_name: '*'
  authorized_networks: 0.0.0.0/0
  ssl_enabled: 'true'
`
	want := make([]CloudSQLRule, 2)
	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
