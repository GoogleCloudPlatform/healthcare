// Package testconf provides utilities to create test configs.
package testconf

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/ghodss/yaml"
)

const configYAML = `
overall:
  organization_id: '12345678'
  folder_id: '98765321'
  billing_account: 000000-000000-000000
  domain: 'my-domain.com'
  allowed_apis:
  - foo-api.googleapis.com
  - bar-api.googleapis.com

forseti:
  project:
    project_id: my-forseti-project
    owners_group: my-forseti-project-owners@my-domain.com
    auditors_group: my-forseti-project-auditors@my-domain.com
    audit_logs:
      logs_bq_dataset:
        properties:
          name: audit_logs
          location: US
      logs_gcs_bucket:
        ttl_days: 365
        properties:
          name: my-forseti-project-logs
          location: US
          storageClass: MULTI_REGIONAL

projects:
- project_id: my-project
  owners_group: my-project-owners@my-domain.com
  auditors_group: my-project-auditors@my-domain.com
  data_readwrite_groups:
  - my-project-readwrite@my-domain.com
  data_readonly_groups:
  - my-project-readonly@my-domain.com
  - another-readonly-group@googlegroups.com
  enabled_apis:
  - foo-api.googleapis.com
  audit_logs:
    logs_bq_dataset:
      properties:
        name: audit_logs
        location: US
    logs_gcs_bucket:
      ttl_days: 365
      properties:
        name: my-project-logs
        location: US
        storageClass: MULTI_REGIONAL
{{lpad .ExtraProjectConfig 2}}

generated_fields:
  projects:
    my-project:
      project_number: '1111'
      log_sink_service_account: audit-logs-bq@logging-1111.iam.gserviceaccount.com
      gce_instance_info:
      - name: foo-instance
        id: '123'
    my-forseti-project:
      project_number: '2222'
      log_sink_service_account: audit-logs-bq@logging-2222.iam.gserviceaccount.com
      gce_instance_info:
      - name: foo-instance
        id: '123'
  forseti:
    service_account: forseti@my-forseti-project.iam.gserviceaccount.com
    server_bucket: gs://my-forseti-project-server/
`

// ConfigData configures a config.
type ConfigData struct {
	ExtraProjectConfig string
}

// ConfigAndProject gets a test config and project.
func ConfigAndProject(t *testing.T, data *ConfigData) (*config.Config, *config.Project) {
	t.Helper()
	if data == nil {
		data = &ConfigData{}
	}

	tmpl, err := template.New("test").Funcs(template.FuncMap{"lpad": lpad}).Parse(configYAML)
	if err != nil {
		t.Fatalf("template Parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template Execute: %v", err)
	}

	validateConfig(t, buf.Bytes())
	conf := new(config.Config)
	if err := yaml.Unmarshal(buf.Bytes(), conf); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if err := conf.Init(); err != nil {
		t.Fatalf("conf.Init = %v", err)
	}

	if len(conf.Projects) != 1 {
		t.Fatalf("len(conf.Projects)=%v, want 1", len(conf.Projects))
	}
	proj := conf.Projects[0]
	return conf, proj
}

func validateConfig(t *testing.T, confb []byte) {
	t.Helper()

	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("ioutil.TempFile: %v", err)
	}
	defer func() {
		os.Remove(f.Name())
	}()

	f.Write(confb)
	f.Close()

	// TODO: use gojsonschema
	cmd := exec.Command(
		"deploy/cmd/validate_config/validate_config",
		"--config_path", f.Name(),
		"--schema_path", "deploy/project_config.yaml.schema",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cmd.CombinedOutput: %v, output:\n%v", err, out)
	}
}

func lpad(s string, n int) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		b.WriteString(strings.Repeat(" ", n))
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
