package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestLogSink(t *testing.T) {
	sinkYAML := `
properties:
  sink: foo-sink
  destination: bigquery.googleapis.com/projects/remote-project/datasets/remote_dataset
  filter: 'logName:"logs/cloudaudit.googleapis.com"'
  uniqueWriterIdentity: true
`
	s := &config.LogSink{}
	if err := yaml.Unmarshal([]byte(sinkYAML), s); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := s.Init(); err != nil {
		t.Fatalf("m.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	b, err := yaml.Marshal(s)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}

	// There are no mutations on the sink, so just use the original metric yaml
	// and validate the parsing is correct.
	if err := yaml.Unmarshal([]byte(sinkYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := s.Name(), "foo-sink"; gotName != wantName {
		t.Errorf("m.ResourceName() = %v, want %v", gotName, wantName)
	}
}
