package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestEnabledAPIsRules(t *testing.T) {
	conf, _ := getTestConfigAndProject(t, nil)
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
