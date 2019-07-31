package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestPubsub(t *testing.T) {
	pubsubYAML := `
properties:
  topic: foo-topic
  accessControl:
  - role: roles/pubsub.publisher
    members:
    - 'user:foo@user.com'
  subscriptions:
  - name: foo-subscription
    accessControl:
    - role: roles/pubsub.viewer
      members:
      - 'user:extra-reader@google.com'`

	p := &config.Pubsub{}
	if err := yaml.Unmarshal([]byte(pubsubYAML), p); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := p.Init(); err != nil {
		t.Fatalf("d.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	byt, err := yaml.Marshal(p)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(byt, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(pubsubYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := p.Name(), "foo-topic"; gotName != wantName {
		t.Errorf("d.ResourceName() = %v, want %v", gotName, wantName)
	}
}
