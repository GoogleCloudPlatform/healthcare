package cft

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/deploymentmanager"
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
  auditors_group: my-project-auditors@my-domain.com
  data_readwrite_groups:
  - some-readwrite-group@my-domain.com
  data_readonly_groups:
  - some-readonly-group@my-domain.com
  - another-readonly-group@googlegroups.com
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
`

const auditDeploymentYAML = `
imports:
- path: {{abs "deploy/cft/templates/bigquery/bigquery_dataset.py"}}
- path: {{abs "deploy/cft/templates/gcs_bucket/gcs_bucket.py"}}

resources:
- name: audit_logs
  type: {{abs "deploy/cft/templates/bigquery/bigquery_dataset.py"}}
  properties:
    name: audit_logs
    location: US
    access:
    - groupByEmail: my-project-owners@my-domain.com
      role: OWNER
    - groupByEmail: my-project-auditors@my-domain.com
      role: READER
    - userByEmail: audit-logs-bq@logging-1111.iam.gserviceaccount.com
      role: WRITER
    setDefaultOwner: false
- name: my-project-logs
  type: {{abs "deploy/cft/templates/gcs_bucket/gcs_bucket.py"}}
  properties:
    name: my-project-logs
    location: US
    storageClass: MULTI_REGIONAL
    bindings:
    - role: roles/storage.admin
      members:
      - group:my-project-owners@my-domain.com
    - role: roles/storage.objectCreator
      members:
      - group:cloud-storage-analytics@google.com
    - role: roles/storage.objectViewer
      members:
      - group:my-project-auditors@my-domain.com
    versioning:
      enabled: true
    logging:
      logBucket: my-project-logs
    lifecycle:
      rule:
      - action:
          type: Delete
        condition:
          age: 365
          isLive: true
`

const wantDefaultResourceDeploymentYAML = `
imports:
- path: {{abs "deploy/templates/audit_log_config.py"}}

resources:
- name: audit-logs-to-bigquery
  type: logging.v2.sink
  properties:
    sink: audit-logs-to-bigquery
    destination: bigquery.googleapis.com/projects/my-project/datasets/audit_logs
    filter: 'logName:"logs/cloudaudit.googleapis.com"'
    uniqueWriterIdentity: true
- name: enable-all-audit-log-policies
  type: {{abs "deploy/templates/audit_log_config.py"}}
  properties:
    name: enable-all-audit-log-policies
- name: bigquery-settings-change-count
  type: logging.v2.metric
  properties:
    metric: bigquery-settings-change-count
    description: Count of bigquery permission changes.
    filter: resource.type="bigquery_resource" AND protoPayload.methodName="datasetservice.update"
    metricDescriptor:
      metricKind: DELTA
      valueType: INT64
      unit: '1'
      labels:
      - key: user
        description: Unexpected user
        valueType: STRING
    labelExtractors:
      user: EXTRACT(protoPayload.authenticationInfo.principalEmail)
- name: iam-policy-change-count
  type: logging.v2.metric
  properties:
    metric: iam-policy-change-count
    description: Count of IAM policy changes.
    filter: protoPayload.methodName="SetIamPolicy" OR protoPayload.methodName:".setIamPolicy"
    metricDescriptor:
      metricKind: DELTA
      valueType: INT64
      unit: '1'
      labels:
      - key: user
        description: Unexpected user
        valueType: STRING
    labelExtractors:
      user: EXTRACT(protoPayload.authenticationInfo.principalEmail)
- name: bucket-permission-change-count
  type: logging.v2.metric
  properties:
    metric: bucket-permission-change-count
    description: Count of GCS permissions changes.
    filter: |-
      resource.type=gcs_bucket AND protoPayload.serviceName=storage.googleapis.com AND
      (protoPayload.methodName=storage.setIamPermissions OR protoPayload.methodName=storage.objects.update)
    metricDescriptor:
      metricKind: DELTA
      valueType: INT64
      unit: '1'
      labels:
      - key: user
        description: Unexpected user
        valueType: STRING
    labelExtractors:
      user: EXTRACT(protoPayload.authenticationInfo.principalEmail)
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
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init = %v", err)
	}
	if len(config.Projects) != 1 {
		t.Fatalf("len(config.Projects)=%v, want 1", len(config.Projects))
	}
	proj := config.Projects[0]
	return config, proj
}

func TestDeploy(t *testing.T) {
	tests := []struct {
		name       string
		configData *ConfigData
		want       string
	}{
		{
			name: "bq_dataset",
			configData: &ConfigData{`
resources:
  bq_datasets:
  - properties:
      name: foo-dataset
      location: US`},
			want: `
imports:
- path: {{abs "deploy/cft/templates/bigquery/bigquery_dataset.py"}}
resources:
- name: foo-dataset
  type: {{abs "deploy/cft/templates/bigquery/bigquery_dataset.py"}}
  properties:
    name: foo-dataset
    location: US
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
			name: "gce_instance",
			configData: &ConfigData{`
resources:
  gce_instances:
  - properties:
      name: foo-instance
      diskImage: projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts
      zone: us-east1-a
      machineType: f1-micro`},

			want: `
imports:
- path: {{abs "deploy/cft/templates/instance/instance.py"}}

resources:
- name: foo-instance
  type: {{abs "deploy/cft/templates/instance/instance.py"}}
  properties:
    name: foo-instance
    diskImage: projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts
    zone: us-east1-a
    machineType: f1-micro`,
		},
		{
			name: "gcs_bucket",
			configData: &ConfigData{`
resources:
  gcs_buckets:
  - expected_users:
    - some-expected-user@my-domain.com
    properties:
      name: foo-bucket
      location: us-east1`},
			want: `
imports:
- path: {{abs "deploy/cft/templates/gcs_bucket/gcs_bucket.py"}}

resources:
- name: foo-bucket
  type: {{abs "deploy/cft/templates/gcs_bucket/gcs_bucket.py"}}
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
  type: logging.v2.metric
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
			name: "iam_custom_role",
			configData: &ConfigData{`
resources:
  iam_custom_roles:
  - properties:
      roleId: fooCustomRole
      includedPermissions:
      - iam.roles.get`},
			want: `
imports:
- path: {{abs "deploy/cft/templates/iam_custom_role/project_custom_role.py"}}

resources:
- name: fooCustomRole
  type:  {{abs "deploy/cft/templates/iam_custom_role/project_custom_role.py"}}
  properties:
    roleId: fooCustomRole
    includedPermissions:
    - iam.roles.get`,
		},
		{
			name: "iam_policy",
			configData: &ConfigData{`
resources:
  iam_policies:
  - name: foo-owner-binding
    properties:
      roles:
      - role: roles/owner
        members:
        - group:foo-owner@my-domain.com`},
			want: `
imports:
- path: {{abs "deploy/cft/templates/iam_member/iam_member.py"}}

resources:
- name: foo-owner-binding
  type:  {{abs "deploy/cft/templates/iam_member/iam_member.py"}}
  properties:
   roles:
   - role: roles/owner
     members:
     - group:foo-owner@my-domain.com`,
		},
		{
			name: "pubsub",
			configData: &ConfigData{`
resources:
  pubsubs:
  - properties:
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
- path: {{abs "deploy/cft/templates/pubsub/pubsub.py"}}

resources:
- name: foo-topic
  type: {{abs "deploy/cft/templates/pubsub/pubsub.py"}}
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
			config, project := getTestConfigAndProject(t, tc.configData)

			type upsertCall struct {
				Name       string
				Deployment *deploymentmanager.Deployment
				ProjectID  string
			}

			var got []upsertCall
			upsertDeployment = func(name string, deployment *deploymentmanager.Deployment, projectID string) error {
				got = append(got, upsertCall{name, deployment, projectID})
				return nil
			}

			if err := Deploy(project, config.ProjectForAuditLogs(project)); err != nil {
				t.Fatalf("Deploy: %v", err)
			}

			want := []upsertCall{
				{"data-protect-toolkit-resources", wantResourceDeployment(t, tc.want), project.ID},
				{"data-protect-toolkit-audit-my-project", parseTemplateToDeployment(t, auditDeploymentYAML), project.ID},
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
			}

			// TODO: validate against schema file too
		})
	}
}

func TestGetLogSinkServiceAccount(t *testing.T) {
	const logSinkJSON = `{
		"createTime": "2019-04-15T20:00:16.734389353Z",
		"destination": "bigquery.googleapis.com/projects/my-project/datasets/audit_logs",
		"filter": "logName:\"logs/cloudaudit.googleapis.com\"",
		"name": "audit-logs-to-bigquery",
		"outputVersionFormat": "V2",
		"updateTime": "2019-04-15T20:00:16.734389353Z",
		"writerIdentity": "serviceAccount:p12345-999999@gcp-sa-logging.iam.gserviceaccount.com"
	}`

	cmdCombinedOutput = func(cmd *exec.Cmd) ([]byte, error) {
		args := []string{"gcloud", "logging", "sinks", "describe"}
		if cmp.Equal(cmd.Args[:len(args)], args) {
			return []byte(logSinkJSON), nil
		}
		return nil, fmt.Errorf("unexpected args: %v", cmd.Args)
	}

	_, project := getTestConfigAndProject(t, nil)
	got, err := getLogSinkServiceAccount(project)
	if err != nil {
		t.Fatalf("getLogSinkServiceAccount: %v", err)
	}

	want := "p12345-999999@gcp-sa-logging.iam.gserviceaccount.com"
	if got != want {
		t.Errorf("log sink service account: got %q, want %q", got, want)
	}
}

func wantResourceDeployment(t *testing.T, yamlTemplate string) *deploymentmanager.Deployment {
	t.Helper()
	defaultDeployment := parseTemplateToDeployment(t, wantDefaultResourceDeploymentYAML)
	userDeployment := parseTemplateToDeployment(t, yamlTemplate)
	userDeployment.Imports = append(defaultDeployment.Imports, userDeployment.Imports...)
	userDeployment.Resources = append(defaultDeployment.Resources, userDeployment.Resources...)
	return userDeployment
}

func parseTemplateToDeployment(t *testing.T, yamlTemplate string) *deploymentmanager.Deployment {
	t.Helper()
	tmpl, err := template.New("test-deployment").Funcs(template.FuncMap{"abs": abs}).Parse(yamlTemplate)
	if err != nil {
		t.Fatalf("template Parse: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("tmpl.Execute: %v", err)
	}

	deployment := new(deploymentmanager.Deployment)
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
