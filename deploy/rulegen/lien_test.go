package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestLienRules(t *testing.T) {
	conf, _ := getTestConfigAndProject(t, nil)
	got, err := LienRules(conf)
	if err != nil {
		t.Fatalf("LienRules = %v", err)
	}

	wantYAML := `
- name: Require project deletion liens for all projects.
  mode: required
  resource:
  - type: organization
    resource_ids:
    - '12345678'
  restrictions:
  - resourcemanager.projects.delete
`
	want := make([]LienRule, 1)
	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
