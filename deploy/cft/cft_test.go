package cft

import (
	"bytes"
	"log"
	"os"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

const configYAML = `
overall:
  organization_id: '12345678'
  folder_id: '98765321'
  billing_account: 000000-000000-000000
  domain: 'domain.com'

projects:
- project_id: my-project
  owners_group: my-project-owners@my-domain.com
  editors_group: my-project-editors@mydomain.com
  auditors_group: some-auditors-group@my-domain.com
  data_readwrite_groups:
  - some-readwrite-group@my-domain.com
  data_readonly_groups:
  - some-readonly-group@my-domain.com
  - another-readonly-group@googlegroups.com
  audit_logs:
    logs_gcs_bucket:
      location: US
      storage_class: MULTI_REGIONAL
      ttl_days: 365
    logs_bigquery_dataset:
      location: US
  resources:
    bigquery_datasets:
    - properties:
        name: my-dataset
`

var (
	config  *Config
	project *Project
)

func TestDeploy(t *testing.T) {
	wantDeploymentYAML := `
imports:
- path: {{.ImportPath}}
resources:
- name: my-dataset
  type: {{.ImportPath}}
  properties:
    access:
    - groupByEmail: my-project-owners@my-domain.com
      role: OWNER
    - groupByEmail: some-readwrite-group@my-domain.com
      role: WRITER
    - groupByEmail: some-readonly-group@my-domain.com
      role: READER
    - groupByEmail: another-readonly-group@googlegroups.com
      role: READER
    name: my-dataset
    setDefaultOwner: false
`
	tmpl, err := template.New("test").Parse(wantDeploymentYAML)
	if err != nil {
		t.Fatalf("template Parse: %v", err)
	}

	path, err := getCFTTemplatePath("bigquery_dataset.py")
	if err != nil {
		t.Fatalf("getCFTTemplatePath: %v", err)
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, map[string]interface{}{"ImportPath": path}); err != nil {
		t.Fatalf("tmpl.Execute: %v", err)
	}

	commander := &fakeCommander{
		listDeploymentName: "managed-cloud-foundation-toolkit",
		wantDeploymentCommand: []string{
			"gcloud", "deployment-manager", "deployments", "update", "managed-cloud-foundation-toolkit",
			"--delete-policy", "ABANDON", "--project", project.ID},
	}

	cmdRun = commander.Run
	cmdCombinedOutput = commander.CombinedOutput

	if err := Deploy(project); err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	if err := yaml.Unmarshal(commander.gotConfigFileContents, got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal(buf.Bytes(), want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}
}

func TestMain(m *testing.M) {
	config = new(Config)
	if err := yaml.Unmarshal([]byte(configYAML), config); err != nil {
		log.Fatalf("unmarshal config: %v", err)
	}
	if len(config.Projects) != 1 {
		log.Fatalf("len(config.Projects)=%v, want 1", len(config.Projects))
	}
	project = config.Projects[0]
	os.Exit(m.Run())
}
