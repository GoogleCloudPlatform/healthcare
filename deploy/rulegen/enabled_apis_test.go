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

func TestEnabledAPIsRules(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)
	got, err := EnabledAPIsRules(conf)
	if err != nil {
		t.Fatalf("EnabledAPIsRules = %v", err)
	}

	wantYAML := `
- name: 'Global API whitelist.'
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - '*'
  services:
  - foo-api.googleapis.com
  - bar-api.googleapis.com
- name: 'API whitelist for my-project.'
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-project
  services:
  - foo-api.googleapis.com
`
	want := make([]EnabledAPIsRule, 2)
	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
