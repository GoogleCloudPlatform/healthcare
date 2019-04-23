package cft

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
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

func getTestConfigAndProject(t *testing.T, data *ConfigData) (*Config, *Project) {
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

	config := new(Config)
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

func TestDeploy(t *testing.T) {
	tests := []struct {
		name       string
		configData *ConfigData
		want       string
	}{
		{
			name: "bigquery_dataset",
			configData: &ConfigData{`
resources:
- bigquery_dataset:
    properties:
      name: foo-dataset`},
			want: `
imports:
- path: {{abs "deploy/cft/templates/bigquery_dataset.py"}}
resources:
- name: foo-dataset
  type: {{abs "deploy/cft/templates/bigquery_dataset.py"}}
  properties:
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
			name: "gcs_bucket",
			configData: &ConfigData{`
resources:
- gcs_bucket:
    expected_users:
    - some-expected-user@my-domain.com
    properties:
      name: foo-bucket
      location: us-east1`},
			want: `
imports:
- path: {{abs "deploy/cft/templates/gcs_bucket.py"}}
- path: {{abs "deploy/templates/metric.py"}}

resources:
- name: foo-bucket
  type: {{abs "deploy/cft/templates/gcs_bucket.py"}}
  properties:
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
      enabled: true
    logging:
      logBucket: my-project-logs
- name: unexpected-access-foo-bucket
  type: {{abs "deploy/templates/metric.py"}}
  properties:
    metric: unexpected-access-foo-bucket
    description: Count of unexpected data access to foo-bucket
    metricDescriptor:
      metricKind: DELTA
      valueType: INT64
      unit: '1'
      labels:
      - key: user
        description: Unexpected user
        valueType: STRING
    labelExtractors:
      user: 'EXTRACT(protoPayload.authenticationInfo.principalEmail)'
    filter: |
      resource.type=gcs_bucket AND
      logName=projects/my-project/logs/cloudaudit.googleapis.com%2Fdata_access AND
      protoPayload.resourceName=projects/_/buckets/foo-bucket AND
      protoPayload.status.code!=7 AND
      protoPayload.authenticationInfo.principalEmail!=(some-expected-user@my-domain.com)
  metadata:
    dependsOn:
    - foo-bucket`,
		},
		{
			name: "pubsub",
			configData: &ConfigData{`
resources:
- pubsub:
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
          - 'user:extra-reader@google.com'`},
			want: `
imports:
- path: {{abs "deploy/cft/templates/pubsub.py"}}

resources:
- name: foo-topic
  type: {{abs "deploy/cft/templates/pubsub.py"}}
  properties:
    topic: foo-topic
    accessControl:
    - role: roles/pubsub.publisher
      members:
      - 'user:foo@user.com'
    subscriptions:
    - name: foo-subscription
      accessControl:
      - role: roles/pubsub.editor
        members:
        - 'group:some-readwrite-group@my-domain.com'
      - role: roles/pubsub.viewer
        members:
        - 'group:some-readonly-group@my-domain.com'
        - 'group:another-readonly-group@googlegroups.com'
        - 'user:extra-reader@google.com'`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, project := getTestConfigAndProject(t, tc.configData)

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

			got := new(Deployment)
			if err := yaml.Unmarshal(commander.gotConfigFileContents, &got); err != nil {
				t.Fatal(err)
			}

			want := getWantDeployment(t, tc.want)

			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
			}

			// TODO: validate against schema file too
		})
	}
}

func getWantDeployment(t *testing.T, yamlTemplate string) *Deployment {
	t.Helper()
	tmpl, err := template.New("test-deployment").Funcs(template.FuncMap{"abs": abs}).Parse(yamlTemplate)
	if err != nil {
		t.Fatalf("template Parse: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("tmpl.Execute: %v", err)
	}

	deployment := new(Deployment)
	if err := yaml.Unmarshal(buf.Bytes(), deployment); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	return deployment
}

func abs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		panic(err)
	}
	return a
}
