package config_test

import (
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestUnmarshalAllGeneratedFields(t *testing.T) {
	testYaml := `
projects:
  some-data:
    project_number: '123123123123'
    log_sink_service_account: p123123123123-001111@gcp-sa-logging.iam.gserviceaccount.com
  some-analytics:
    project_number: '456456456456'
    log_sink_service_account: p456456456456-002222@gcp-sa-logging.iam.gserviceaccount.com
    gce_instance_info:
    - name: foo-instance
      id: '123'
forseti:
  service_account: some-forseti-gcp-reader@some-forseti.iam.gserviceaccount.com
  server_bucket: gs://some-forseti-server
`
	got := new(config.AllGeneratedFields)
	yaml.Unmarshal([]byte(testYaml), got)
	if err := yaml.Unmarshal([]byte(testYaml), got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	want := &config.AllGeneratedFields{
		Projects: map[string]*config.GeneratedFields{
			"some-data": &config.GeneratedFields{
				ProjectNumber:         "123123123123",
				LogSinkServiceAccount: "p123123123123-001111@gcp-sa-logging.iam.gserviceaccount.com",
			},
			"some-analytics": &config.GeneratedFields{
				ProjectNumber:         "456456456456",
				LogSinkServiceAccount: "p456456456456-002222@gcp-sa-logging.iam.gserviceaccount.com",
				GCEInstanceInfoList:   []config.GCEInstanceInfo{{Name: "foo-instance", ID: "123"}}},
		},
		Forseti: config.ForsetiServiceInfo{
			ServiceAccount: "some-forseti-gcp-reader@some-forseti.iam.gserviceaccount.com",
			ServiceBucket:  "gs://some-forseti-server",
		},
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("AllGeneratedFields mismatch (-want +got):\n%s", diff)
	}
}

func TestGetInstanceID(t *testing.T) {
	const projectID = "some-analytics"
	gf := &config.AllGeneratedFields{
		Projects: map[string]*config.GeneratedFields{
			projectID: &config.GeneratedFields{
				GCEInstanceInfoList: []config.GCEInstanceInfo{
					{Name: "foo-instance", ID: "123"},
					{Name: "bar-instance", ID: "456"},
				},
			},
		},
	}

	project, ok := gf.Projects[projectID]
	if !ok {
		t.Fatalf("missing %q in %v", projectID, gf.Projects)
	}

	name := "foo-instance"
	if id, err := project.InstanceID(name); err != nil {
		t.Errorf("project.InstanceID(%q): got error %v", name, err)
	} else {
		if id != "123" {
			t.Errorf("project.InstanceID(%q) id: got %q, want 123", name, id)
		}
	}

	name = "dne"
	if _, err := gf.Projects["some-analytics"].InstanceID(name); err == nil {
		t.Errorf("project.InstanceID(%q): got nil error, want non-nil error", name)
	}
}
