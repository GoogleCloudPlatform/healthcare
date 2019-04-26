package cft

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestGCEInstance(t *testing.T) {
	_, project := getTestConfigAndProject(t, nil)

	instanceYAML := `
properties:
  name: foo-instance
  zone: us-east1-a
`

	ins := &GCEInstance{}
	if err := yaml.Unmarshal([]byte(instanceYAML), ins); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := ins.Init(project); err != nil {
		t.Fatalf("d.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	byt, err := yaml.Marshal(ins)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(byt, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(instanceYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := ins.Name(), "foo-instance"; gotName != wantName {
		t.Errorf("ins.Name() = %v, want %v", gotName, wantName)
	}
}
