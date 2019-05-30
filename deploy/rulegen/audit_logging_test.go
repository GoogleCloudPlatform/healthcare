package rulegen

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestAuditLoggingRules(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)
	got, err := AuditLoggingRules(conf)
	if err != nil {
		t.Fatalf("AuditLoggingRules = %v", err)
	}

	wantYAML := `
- name: Require all Cloud Audit logs.
  resource:
  - type: project
    resource_ids:
    - '*'
  service: allServices
  log_types:
  - ADMIN_READ
  - DATA_READ
  - DATA_WRITE
`
	want := make([]AuditLoggingRule, 1)
	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
