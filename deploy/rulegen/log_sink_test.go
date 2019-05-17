package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestLogSinkRules(t *testing.T) {
	config, _ := getTestConfigAndProject(t, nil)
	got, err := LogSinkRules(config)
	if err != nil {
		t.Fatalf("LogSinkRules = %v", err)
	}

	wantYAML := `
- name: 'Require a BigQuery Log sink in all projects.'
  mode: required
  resource:
  - type: organization
    applies_to: children
    resource_ids:
    - '12345678'
  sink:
    destination: 'bigquery.googleapis.com/*'
    filter: '*'
    include_children: '*'
- name: 'Only allow BigQuery Log sinks in all projects.'
  mode: whitelist
  resource:
  - type: organization
    applies_to: children
    resource_ids:
    - '12345678'
  sink:
    destination: 'bigquery.googleapis.com/*'
    filter: '*'
    include_children: '*'
- name: 'Require Log sink for project my-forseti-project.'
  mode: required
  resource:
  - type: project
    applies_to: self
    resource_ids:
    - my-forseti-project
  sink:
    destination: >-
      bigquery.googleapis.com/projects/my-forseti-project/datasets/audit_logs
    filter: 'logName:"logs/cloudaudit.googleapis.com"'
    include_children: '*'
- name: 'Whitelist Log sink for project my-forseti-project.'
  mode: whitelist
  resource:
  - type: project
    applies_to: self
    resource_ids:
    - my-forseti-project
  sink:
    destination: >-
      bigquery.googleapis.com/projects/my-forseti-project/datasets/audit_logs
    filter: 'logName:"logs/cloudaudit.googleapis.com"'
    include_children: '*'
- name: 'Require Log sink for project my-project.'
  mode: required
  resource:
  - type: project
    applies_to: self
    resource_ids:
    - my-project
  sink:
    destination: >-
      bigquery.googleapis.com/projects/my-project/datasets/audit_logs
    filter: 'logName:"logs/cloudaudit.googleapis.com"'
    include_children: '*'
- name: 'Whitelist Log sink for project my-project.'
  mode: whitelist
  resource:
  - type: project
    applies_to: self
    resource_ids:
    - my-project
  sink:
    destination: >-
      bigquery.googleapis.com/projects/my-project/datasets/audit_logs
    filter: 'logName:"logs/cloudaudit.googleapis.com"'
    include_children: '*'
`
	want := make([]LogSinkRule, 0)
	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
