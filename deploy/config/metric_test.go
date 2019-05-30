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
