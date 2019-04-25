package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestBucket(t *testing.T) {
	config, _ := getTestConfigAndProject(t, nil)
	got, err := BucketRules(config)
	if err != nil {
		t.Fatalf("BucketRules = %v", err)
	}

	wantYAML := `
- name: Disallow all acl rules, only allow IAM.
  bucket: '*'
  entity: '*'
  email: '*'
  domain: '*'
  role: '*'
  resource:
  - resource_ids:
    - '*'
`
	want := make([]BucketRule, 1)
	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
