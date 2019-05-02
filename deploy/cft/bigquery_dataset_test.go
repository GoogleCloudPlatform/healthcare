package cft

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestDataset(t *testing.T) {
	_, project := getTestConfigAndProject(t, nil)

	datasetYAML := `
properties:
  name: foo-dataset
  location: US
  access:
  - userByEmail: some-admin@domain.com
    role: OWNER
`

	wantdatasetYAML := `
properties:
  name: foo-dataset
  location: US
  access:
  - userByEmail: some-admin@domain.com
    role: OWNER
  - groupByEmail: my-project-owners@my-domain.com
    role: OWNER
  - groupByEmail: some-readwrite-group@my-domain.com
    role: WRITER
  - groupByEmail: some-readonly-group@my-domain.com
    role: READER
  - groupByEmail: another-readonly-group@googlegroups.com
    role: READER
  setDefaultOwner: false
`

	d := new(BigqueryDataset)
	if err := yaml.Unmarshal([]byte(datasetYAML), d); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := d.Init(project); err != nil {
		t.Fatalf("d.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	b, err := yaml.Marshal(d)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(wantdatasetYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := d.Name(), "foo-dataset"; gotName != wantName {
		t.Errorf("d.ResourceName() = %v, want %v", gotName, wantName)
	}
}

func TestDatasetErrors(t *testing.T) {
	_, project := getTestConfigAndProject(t, nil)

	tests := []struct {
		name string
		yaml string
		err  string
	}{
		{
			"missing_name",
			"properties: {}",
			"name must be set",
		},
		{
			"missing_location",
			"properties: {name: foo-dataset}",
			"location must be set",
		},
		{
			"set_default_owner",
			"properties: {name: foo-dataset, location: US, setDefaultOwner: true}",
			"setDefaultOwner must not be true",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := new(BigqueryDataset)
			if err := yaml.Unmarshal([]byte(tc.yaml), d); err != nil {
				t.Fatalf("yaml unmarshal: %v", err)
			}
			if err := d.Init(project); err == nil {
				t.Fatalf("d.Init error: got nil, want %v", tc.err)
			} else if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("d.Init: got error %q, want error with substring %q", err, tc.err)
			}
		})
	}
}
