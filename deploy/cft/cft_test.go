package cft

import (
	"bytes"
	"strings"
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
{{lpad .ExtraProjectConfig 2}}
`

type ConfigOptions struct {
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

func getTestConfigAndProject(t *testing.T, opts *ConfigOptions) (*Config, *Project) {
	if opts == nil {
		opts = &ConfigOptions{}
	}

	tmpl, err := template.New("test").Funcs(template.FuncMap{"lpad": lpad}).Parse(configYAML)
	if err != nil {
		t.Fatalf("template Parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, opts); err != nil {
		t.Fatalf("template Execute: %v", err)
	}

	config := new(Config)
	if err := yaml.Unmarshal(buf.Bytes(), config); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if len(config.Projects) != 1 {
		t.Fatalf("len(config.Projects)=%v, want 1", len(config.Projects))
	}
	return config, config.Projects[0]
}

func TestDeploy(t *testing.T) {
	tests := []struct {
		name                     string
		resourceName             string
		templatePath             string
		configOpts               *ConfigOptions
		wantDeploymentProperties string
	}{
		{
			name:         "bigquery_dataset",
			resourceName: "foo-dataset",
			templatePath: "bigquery_dataset.py",
			configOpts: &ConfigOptions{`
resources:
- bigquery_dataset:
    properties:
      name: foo-dataset`},
			wantDeploymentProperties: `
name: foo-dataset
access:
- groupByEmail: my-project-owners@my-domain.com
  role: OWNER
- groupByEmail: some-readwrite-group@my-domain.com
  role: WRITER
- groupByEmail: some-readonly-group@my-domain.com
  role: READER
- groupByEmail: another-readonly-group@googlegroups.com
  role: READER
setDefaultOwner: false`,
		},
		{
			name:         "gcs_bucket",
			resourceName: "foo-bucket",
			templatePath: "gcs_bucket.py",
			configOpts: &ConfigOptions{`
resources:
- gcs_bucket:
    no_prefix: true
    properties:
      name: foo-bucket
      location: us-east1`},
			wantDeploymentProperties: `
name: foo-bucket
location: us-east1
bindings:
- role: roles/storage.admin
  members:
  - 'group:my-project-owners@my-domain.com'
- role: roles/storage.objectAdmin
  members:
  - 'group:some-readwrite-group@my-domain.com'
- role: roles/storage.objectViewer
  members:
  - 'group:some-readonly-group@my-domain.com'
  - 'group:another-readonly-group@googlegroups.com'
versioning:
  enabled: true`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, project := getTestConfigAndProject(t, tc.configOpts)

			wantDeploymentYAML := `
imports:
- path: {{.ImportPath}}
resources:
- name: {{.Name}}
  type: {{.Type}}
  properties:`
			wantDeploymentYAML += lpad(tc.wantDeploymentProperties, 4)

			tmpl, err := template.New("test").Parse(wantDeploymentYAML)
			if err != nil {
				t.Fatalf("template Parse: %v", err)
			}

			path, err := getCFTTemplatePath(tc.templatePath)
			if err != nil {
				t.Fatalf("getCFTTemplatePath: %v", err)
			}

			var buf bytes.Buffer
			data := struct {
				Name, ImportPath, Type string
			}{
				tc.resourceName, path, tc.templatePath,
			}
			if err := tmpl.Execute(&buf, data); err != nil {
				t.Fatalf("tmpl.Execute: %v", err)
			}

			commander := &fakeCommander{
				listDeploymentName: "managed-data-protect-toolkit",
				wantDeploymentCommand: []string{
					"gcloud", "deployment-manager", "deployments", "update", "managed-data-protect-toolkit",
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

			// TODO: validate against schema file too
		})
	}
}
