package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestIAMCustomRole(t *testing.T) {
	customRoleYAML := `
properties:
  roleId: fooCustomRole
`
	i := &config.IAMCustomRole{}
	if err := yaml.Unmarshal([]byte(customRoleYAML), i); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := i.Init(); err != nil {
		t.Fatalf("m.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	b, err := yaml.Marshal(i)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}

	// There are no mutations on the custom role, so just use the original yaml
	// and validate the parsing is correct.
	if err := yaml.Unmarshal([]byte(customRoleYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := i.Name(), "fooCustomRole"; gotName != wantName {
		t.Errorf("m.ResourceName() = %v, want %v", gotName, wantName)
	}
}

func TestIAMPolicy(t *testing.T) {
	customRoleYAML := `
name: foo-owner-binding
properties:
  roles:
  - role: roles/owner
    members:
    - group:foo-owner@my-domain.com
`
	i := &config.IAMPolicy{}
	if err := yaml.Unmarshal([]byte(customRoleYAML), i); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := i.Init(); err != nil {
		t.Fatalf("m.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	b, err := yaml.Marshal(i)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}

	// There are no mutations on the custom role, so just use the original yaml
	// and validate the parsing is correct.
	if err := yaml.Unmarshal([]byte(customRoleYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := i.Name(), "foo-owner-binding"; gotName != wantName {
		t.Errorf("m.ResourceName() = %v, want %v", gotName, wantName)
	}
}
