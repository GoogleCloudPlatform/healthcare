package cft

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

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
- path: {{abs "deploy/cft/templates/iam_member/iam_member.py"}}

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
- name: required-project-bindings
  type: {{abs "deploy/cft/templates/iam_member/iam_member.py"}}
  properties:
    roles:
    - role: roles/owner
      members:
      - group:my-project-owners@my-domain.com
    - role: roles/iam.securityReviewer
      members:
      - group:my-project-auditors@my-domain.com
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
