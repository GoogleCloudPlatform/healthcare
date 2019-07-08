package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestTerraformModule(t *testing.T) {
	got := make(map[string]interface{})
	want := map[string]interface{}{
		"source":     "foo",
		"prop-field": "bar",
	}
	b, err := yaml.Marshal(&config.TerraformModule{
		Source:     "foo",
		Properties: map[string]interface{}{"prop-field": "bar"},
	})
	if err != nil {
		t.Fatalf("yaml.Marshal properties: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}
}
