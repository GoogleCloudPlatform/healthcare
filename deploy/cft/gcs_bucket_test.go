package cft

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestGCSBucket(t *testing.T) {
	_, project := getTestConfigAndProject(t, nil)

	bucketYAML := `
properties:
  name: foo-bucket
  bindings:
  - role: roles/storage.objectViewer
    members:
    - 'user:extra-reader@google.com'
`

	wantBucketYAML := `
properties:
  name: foo-bucket
  bindings:
  - role: roles/storage.admin
    members:
    - 'group:my-project-owners@my-domain.com'
  - role: roles/storage.objectAdmin
    members:
    - 'group:some-readwrite-group@my-domain.com'
  - role: roles/storage.objectViewer
    members:
    - 'group:some-readonly-group@my-domain.com'
    - 'group:another-readonly-group@googlegroups.com'
    - 'user:extra-reader@google.com'
  versioning:
    enabled: True
`

	b := &GCSBucket{}
	if err := yaml.Unmarshal([]byte(bucketYAML), b); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := b.Init(project); err != nil {
		t.Fatalf("d.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	byt, err := yaml.Marshal(b)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(byt, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(wantBucketYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := b.Name(), "foo-bucket"; gotName != wantName {
		t.Errorf("d.ResourceName() = %v, want %v", gotName, wantName)
	}
}

func TestGCSBucketErrors(t *testing.T) {
	_, project := getTestConfigAndProject(t, nil)

	tests := []struct {
		name string
		yaml string
		err  string
	}{
		{
			"missing_name",
			`properties	: {}`,
			"name must be set",
		}, {
			"versioning_disabled",
			`properties: { name: foo-bucket, versioning: { enabled: false }}`,
			"versioning must not be disabled",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(*testing.T) {
			b := &GCSBucket{}
			if err := yaml.Unmarshal([]byte(tc.yaml), b); err != nil {
				t.Fatalf("yaml unmarshal: %v", err)
			}
			if err := b.Init(project); err == nil {
				t.Fatalf("b.Init error: got nil, want %v", tc.err)
			} else if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("b.Init: got error %q, want error with substring %q", err, tc.err)
			}
		})
	}
}
