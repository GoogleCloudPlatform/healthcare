package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestMetric(t *testing.T) {
	_, project := testconf.ConfigAndProject(t, nil)

	metricYAML := `
properties:
  metric: access-buckets-failures
  description: Count of failed access buckets
  filter: 'resource.type=gcs_bucket AND protoPayload.status.code != 0'
  metricDescriptor:
    metricKind: DELTA
    valueType: INT64
    unit: '1'
    labels:
    - key: user
      valueType: STRING
      description: Denied user
  labelExtractors:
    user: 'EXTRACT(protoPayload.authenticationInfo.principalEmail)'
`
	m := &config.Metric{}
	if err := yaml.Unmarshal([]byte(metricYAML), m); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := m.Init(project); err != nil {
		t.Fatalf("m.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	b, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}

	// There are no mutations on the metric, so just use the original metric yaml
	// and validate the parsing is correct.
	if err := yaml.Unmarshal([]byte(metricYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := m.Name(), "access-buckets-failures"; gotName != wantName {
		t.Errorf("m.ResourceName() = %v, want %v", gotName, wantName)
	}
}

func TestViolationExceptions(t *testing.T) {
	conf := testconf.ConfigBeforeInit(t, nil)

	conf.Projects[0].ViolationExceptions = make(map[string][]string)
	conf.Projects[0].ViolationExceptions["iam-policy-change-count"] = []string{
		"some-account1@domain.com",
		"some-account2@domain.com",
	}
	conf.Projects[0].ViolationExceptions["some-not-exist-metrics"] = []string{
		"some-account9@domain.com",
		"some-account8@domain.com",
	}
	if err := conf.Projects[0].Init(conf.AuditLogsProject); err != nil {
		t.Fatalf("failed to init project %q: %v", conf.Projects[0].ID, err)
	}
	expectedIAMPolicyChangeCountFilter := `protoPayload.methodName="SetIamPolicy" OR protoPayload.methodName:".setIamPolicy" AND
protoPayload.authenticationInfo.principalEmail!=(some-account1@domain.com AND some-account2@domain.com)`
	for _, m := range conf.Projects[0].Metrics {
		if m.MetricProperties.MetricName == "iam-policy-change-count" {
			if diff := cmp.Diff(m.Filter, expectedIAMPolicyChangeCountFilter); diff != "" {
				t.Fatalf("yaml differs (-got +want):\n%v", diff)
			}
		}
	}
}

func TestDoNotHaveViolationExceptions(t *testing.T) {
	conf := testconf.ConfigBeforeInit(t, nil)

	conf.Projects[0].ViolationExceptions = make(map[string][]string)
	if err := conf.Projects[0].Init(conf.AuditLogsProject); err != nil {
		t.Fatalf("failed to init project %q: %v", conf.Projects[0].ID, err)
	}
	expectedIAMPolicyChangeCountFilter := `protoPayload.methodName="SetIamPolicy" OR protoPayload.methodName:".setIamPolicy"`
	for _, m := range conf.Projects[0].Metrics {
		if m.MetricProperties.MetricName == "iam-policy-change-count" {
			if diff := cmp.Diff(m.Filter, expectedIAMPolicyChangeCountFilter); diff != "" {
				t.Fatalf("yaml differs (-got +want):\n%v", diff)
			}
		}
	}
}
