package cft

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestUnmarshalAllGeneratedFields(t *testing.T) {
	testYaml := `
generated_fields:
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
	cfg := &Config{}
	yaml.Unmarshal([]byte(testYaml), cfg)
	if err := yaml.Unmarshal([]byte(testYaml), cfg); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	expectedConfig := Config{}
	expectedConfig.AllOfGeneratedFields.Projects = make(map[string]*GeneratedFields)
	expectedConfig.AllOfGeneratedFields.Projects["some-data"] = &GeneratedFields{ProjectNumber: "123123123123", LogSinkServiceAccount: "p123123123123-001111@gcp-sa-logging.iam.gserviceaccount.com"}
	expectedConfig.AllOfGeneratedFields.Projects["some-analytics"] = &GeneratedFields{ProjectNumber: "456456456456", LogSinkServiceAccount: "p456456456456-002222@gcp-sa-logging.iam.gserviceaccount.com", GCEInstanceInfoList: []GCEInstanceInfo{{Name: "foo-instance", ID: "123"}}}
	expectedConfig.AllOfGeneratedFields.Forseti = ForsetiServiceInfo{ServiceAccount: "some-forseti-gcp-reader@some-forseti.iam.gserviceaccount.com", ServiceBucket: "gs://some-forseti-server"}

	if diff := cmp.Diff(*cfg, expectedConfig); diff != "" {
		t.Fatalf("AllGeneratedFields mismatch (-want +got):\n%s", diff)
	}
}

func TestGetInstanceID(t *testing.T) {
	cfg := Config{}
	cfg.AllOfGeneratedFields.Projects = make(map[string]*GeneratedFields)
	cfg.AllOfGeneratedFields.Projects["some-analytics"] = &GeneratedFields{GCEInstanceInfoList: []GCEInstanceInfo{{Name: "foo-instance", ID: "123"}, {Name: "bar-instance", ID: "456"}}}

	project, err := cfg.AllOfGeneratedFields.Project("some-analytics")
	if err != nil {
		t.Fatalf("cfg.AllOfGeneratedFields.Project(\"some-analytics\") = %v", err)
	}

	name := "foo-instance"
	if id, err := project.InstanceID(name); err != nil {
		t.Errorf("project.InstanceID(%q): got error %v", name, err)
	} else {
		if id != "123" {
			t.Errorf("project.InstanceID(%q) id: got %q, want 123", name, id)
		}
	}

	_, err = cfg.AllOfGeneratedFields.Project("none-analytics")
	if err == nil {
		t.Errorf("AllOfGeneratedFields.Project: got nil error, want non-nil error")
	}

	name = "dne"
	if _, err := cfg.AllOfGeneratedFields.Projects["some-analytics"].InstanceID(name); err == nil {
		t.Errorf("project.InstanceID(%q): got nil error, want non-nil error", name)
	}
}

func TestGetProjectInGeneratedFields(t *testing.T) {
	cfg := Config{}
	cfg.AllOfGeneratedFields.Projects = make(map[string]*GeneratedFields)
	cfg.AllOfGeneratedFields.Projects["some-data"] = &GeneratedFields{ProjectNumber: "11111"}
	expectProjectNumber := "22222"
	cfg.AllOfGeneratedFields.Projects["some-analytics"] = &GeneratedFields{ProjectNumber: expectProjectNumber}
	project, err := cfg.AllOfGeneratedFields.Project("some-analytics")
	if project.ProjectNumber != expectProjectNumber {
		t.Errorf("get project error: got %v, want %q, err %v", project.ProjectNumber, expectProjectNumber, err)
	}
}
