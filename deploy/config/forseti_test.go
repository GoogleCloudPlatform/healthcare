package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestForseti(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	var got interface{}
	want := map[string]interface{}{
		"project_id": "my-forseti-project",
		"domain":     "my-domain.com",
		"composite_root_resources": []interface{}{
			"organizations/12345678",
			"folders/98765321",
		},
		"storage_bucket_location": "us-east1",
	}
	b, err := yaml.Marshal(conf.Forseti.Properties)
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
