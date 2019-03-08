package cft

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestDataset(t *testing.T) {
	datasetYAML := `
cft:
  name: foo-dataset
  access:
  - userByEmail: some-admin@domain.com
    role: OWNER
  - specialGroup: allAuthenticatedUsers
    role: READER
`

	wantdatasetYAML := `
cft:
  name: foo-dataset
  access:
  - userByEmail: some-admin@domain.com
    role: OWNER
  - specialGroup: allAuthenticatedUsers
    role: READER
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

	d, err := NewBigqueryDataset(project, func(d *BigqueryDataset) error {
		if err := yaml.Unmarshal([]byte(datasetYAML), d); err != nil {
			return fmt.Errorf("test yaml unmarshal: %v", err)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("NewDataset: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	b, err := yaml.Marshal(d)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(b, got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(wantdatasetYAML), want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := d.ResourceName(), "foo-dataset"; gotName != wantName {
		t.Errorf("d.ResourceName() = %v, want %v", gotName, wantName)
	}
}

func TestDatasetErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		err  string
	}{
		{
			"missing_name",
			`"cft": {}`,
			"name must be set",
		},
		{
			"set_default_owner",
			`"cft": {"name": "foo-dataset", "setDefaultOwner": true}`,
			"setDefaultOwner must not be true",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(*testing.T) {
			setYaml := func(d *BigqueryDataset) error {
				if err := yaml.Unmarshal([]byte(tc.yaml), d); err != nil {
					return fmt.Errorf("test yaml unmarshal: %v", err)
				}
				return nil
			}
			if _, err := NewBigqueryDataset(project, setYaml); err == nil {
				t.Fatalf("NewBigqueryDataset expected error: got nil, want %v", tc.err)
			} else if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("NewBigqueryDataset: got error %q, want error with substring %q", err, tc.err)
			}
		})
	}
}
