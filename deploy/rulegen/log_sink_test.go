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

func TestLogSinkRules(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)
	got, err := LogSinkRules(conf)
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
