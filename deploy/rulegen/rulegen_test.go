package rulegen

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

// TODO: This is copied from cft_test.go. Pull out into own package.
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
  editors_group: my-project-editors@my-domain.com
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

	tmpl, err := template.New("").Funcs(template.FuncMap{"lpad": lpad}).Parse(configYAML)
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
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init = %v", err)
	}
	if len(config.Projects) != 1 {
		t.Fatalf("len(config.Projects)=%v, want 1", len(config.Projects))
	}
	proj := config.Projects[0]
	return config, proj
}

func TestRunOutputPath(t *testing.T) {
	config, _ := getTestConfigAndProject(t, nil)

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("ioutil.TempDir = %v", err)
	}

	if err := Run(config, tmpDir); err != nil {
		t.Fatalf("Run = %v", err)
	}

	checkRulesDir(t, tmpDir)
}

func TestRunServerBucket(t *testing.T) {
	config, _ := getTestConfigAndProject(t, nil)

	var gotArgs []string
	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		if len(gotArgs) != 0 {
			return nil, errors.New("fake CombinedOutput: unexpectedly called more than once")
		}

		gotArgs = cmd.Args
		if len(gotArgs) != 4 {
			return nil, fmt.Errorf("fake CombinedOutput: unexpected number of args: got %d, want 4, %v ", len(cmd.Args), cmd.Args)
		}
		checkRulesDir(t, filepath.Dir(gotArgs[2]))
		return nil, nil
	}
	if err := Run(config, ""); err != nil {
		t.Fatalf("Run = %v", err)
	}

	wantRE, err := regexp.Compile(`gsutil cp .*\*\.yaml gs://my-forseti-project-server/rules`)
	if err != nil {
		t.Fatalf("regexp.Compile = %v", err)
	}
	got := strings.Join(gotArgs, " ")
	if !wantRE.MatchString(got) {
		t.Fatalf("rules upload command does not match: got %q, want match of %q", got, wantRE)
	}
}

func checkRulesDir(t *testing.T, rulesDir string) {
	t.Helper()

	// check one rules file
	b, err := ioutil.ReadFile(filepath.Join(rulesDir, "audit_logging_rules.yaml"))
	if err != nil {
		t.Fatalf("ioutil.ReadFile = %v", err)
	}

	wantYAML := `
rules:
- name: Require all Cloud Audit logs.
  resource:
  - type: project
    resource_ids:
    - '*'
  service: allServices
  log_types:
  - ADMIN_READ
  - DATA_READ
  - DATA_WRITE
`
	got := make(map[string]interface{})
	want := make(map[string]interface{})

	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("audit logging rules differ (-got, +want):\n%v", diff)
	}
}
