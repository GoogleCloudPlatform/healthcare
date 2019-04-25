package rulegen

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"github.com/ghodss/yaml"
)

// TODO: This is copied from cft_test.go. Pull out into own package.
const configYAML = `
overall:
  organization_id: '12345678'
  folder_id: '98765321'
  billing_account: 000000-000000-000000
  domain: 'domain.com'
  allowed_apis:
  - foo-api.googleapis.com
  - bar-api.googleapis.com

projects:
- project_id: my-project
  owners_group: my-project-owners@my-domain.com
  editors_group: my-project-editors@mydomain.com
  auditors_group: my-project-auditors@my-domain.com
  data_readwrite_groups:
  - my-project-readwrite@my-domain.com
  data_readonly_groups:
  - my-project-readonly@my-domain.com
  - another-readonly-group@googlegroups.com
  enabled_apis:
  - foo-api.googleapis.com
  audit_logs:
    logs_gcs_bucket:
      location: US
      storage_class: MULTI_REGIONAL
      ttl_days: 365
    logs_bigquery_dataset:
      location: US
  generated_fields:
    project_number: '1111'
    log_sink_service_account: audit-logs-bq@logging-1111.iam.gserviceaccount.com
{{lpad .ExtraProjectConfig 2}}
`

type ConfigData struct {
	ExtraProjectConfig string
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

func getTestConfigAndProject(t *testing.T, data *ConfigData) (*cft.Config, *cft.Project) {
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

	config := new(cft.Config)
	if err := yaml.Unmarshal(buf.Bytes(), config); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if len(config.Projects) != 1 {
		t.Fatalf("len(config.Projects)=%v, want 1", len(config.Projects))
	}
	proj := config.Projects[0]
	if err := proj.Init(); err != nil {
		t.Fatalf("proj.Init: %v", err)
	}
	return config, proj
}
