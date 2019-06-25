package apply

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/deploymentmanager"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ghodss/yaml"
)

const auditDeploymentYAML = `
imports:
- path: {{abs "deploy/config/templates/bigquery/bigquery_dataset.py"}}
- path: {{abs "deploy/config/templates/gcs_bucket/gcs_bucket.py"}}

resources:
- name: audit_logs
  type: {{abs "deploy/config/templates/bigquery/bigquery_dataset.py"}}
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
  type: {{abs "deploy/config/templates/gcs_bucket/gcs_bucket.py"}}
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
- path: {{abs "deploy/config/templates/iam_member/iam_member.py"}}

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
  type: {{abs "deploy/config/templates/iam_member/iam_member.py"}}
  properties:
    roles:
    - role: roles/owner
      members:
      - group:my-project-owners@my-domain.com
    - role: roles/iam.securityReviewer
      members:
      - group:my-project-auditors@my-domain.com
`

func TestDeploy(t *testing.T) {
	cmdRun = func(cmd *exec.Cmd) error { return nil }
	cmdOutput = func(cmd *exec.Cmd) ([]byte, error) {
		args := strings.Join(cmd.Args, " ")
		var res string
		switch {
		case strings.HasPrefix(args, "gcloud config get-value account"):
			res = `"foo-user@my-domain.com"`
		case strings.HasPrefix(args, "gcloud projects get-iam-policy"):
			res = "{}"
		default:
			return nil, fmt.Errorf("unexpected args: %v", args)
		}
		return []byte(res), nil
	}

	tests := []struct {
		name       string
		configData *testconf.ConfigData
		want       string
	}{
		{
			name: "bq_dataset",
			configData: &testconf.ConfigData{`
resources:
  bq_datasets:
  - properties:
      name: foo-dataset
      location: US`},
			want: `
imports:
- path: {{abs "deploy/config/templates/bigquery/bigquery_dataset.py"}}
resources:
- name: foo-dataset
  type: {{abs "deploy/config/templates/bigquery/bigquery_dataset.py"}}
  properties:
    name: foo-dataset
    location: US
    access:
    - groupByEmail: my-project-owners@my-domain.com
      role: OWNER
    - groupByEmail: my-project-readwrite@my-domain.com
      role: WRITER
    - groupByEmail: my-project-readonly@my-domain.com
      role: READER
    - groupByEmail: another-readonly-group@googlegroups.com
      role: READER
    setDefaultOwner: false`,
		},
		{
			name: "gce_firewall",
			configData: &testconf.ConfigData{`
resources:
  gce_firewalls:
  - properties:
      name: foo-firewall-rules
      network: foo-network
      rules:
      - name: allow-proxy-from-inside
        allowed:
        - IPProtocol: tcp
          ports:
          - "80"
          - "443"
          description: test rule for network-test
          direction: INGRESS
          sourceRanges:
          - 10.0.0.0/8`},
			want: `
imports:
- path: {{abs "deploy/config/templates/firewall/firewall.py"}}

resources:
- name: foo-firewall-rules
  type: {{abs "deploy/config/templates/firewall/firewall.py"}}
  properties:
    name: foo-firewall-rules
    network: foo-network
    rules:
    - name: allow-proxy-from-inside
      allowed:
      - IPProtocol: tcp
        ports:
        - "80"
        - "443"
        description: test rule for network-test
        direction: INGRESS
        sourceRanges:
        - 10.0.0.0/8`,
		},
		{
			name: "gce_instance",
			configData: &testconf.ConfigData{`
resources:
  gce_instances:
  - properties:
      name: foo-instance
      diskImage: projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts
      zone: us-east1-a
      machineType: f1-micro`},
			want: `
imports:
- path: {{abs "deploy/config/templates/instance/instance.py"}}

resources:
- name: foo-instance
  type: {{abs "deploy/config/templates/instance/instance.py"}}
  properties:
    name: foo-instance
    diskImage: projects/ubuntu-os-cloud/global/images/family/ubuntu-1804-lts
    zone: us-east1-a
    machineType: f1-micro`,
		},
		{
			name: "gcs_bucket",
			configData: &testconf.ConfigData{`
resources:
  gcs_buckets:
  - expected_users:
    - some-expected-user@my-domain.com
    properties:
      name: foo-bucket
      location: us-east1`},
			want: `
imports:
- path: {{abs "deploy/config/templates/gcs_bucket/gcs_bucket.py"}}

resources:
- name: foo-bucket
  type: {{abs "deploy/config/templates/gcs_bucket/gcs_bucket.py"}}
  properties:
    name: foo-bucket
    location: us-east1
    bindings:
    - role: roles/storage.admin
      members:
      - 'group:my-project-owners@my-domain.com'
    - role: roles/storage.objectAdmin
      members:
      - 'group:my-project-readwrite@my-domain.com'
    - role: roles/storage.objectViewer
      members:
      - 'group:my-project-readonly@my-domain.com'
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
    filter: |-
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
			configData: &testconf.ConfigData{`
resources:
  iam_custom_roles:
  - properties:
      roleId: fooCustomRole
      includedPermissions:
      - iam.roles.get`},
			want: `
imports:
- path: {{abs "deploy/config/templates/iam_custom_role/project_custom_role.py"}}

resources:
- name: fooCustomRole
  type:  {{abs "deploy/config/templates/iam_custom_role/project_custom_role.py"}}
  properties:
    roleId: fooCustomRole
    includedPermissions:
    - iam.roles.get`,
		},
		{
			name: "iam_policy",
			configData: &testconf.ConfigData{`
resources:
  iam_policies:
  - name: foo-owner-binding
    properties:
      roles:
      - role: roles/owner
        members:
        - group:foo-owner@my-domain.com`},
			want: `
resources:
- name: foo-owner-binding
  type:  {{abs "deploy/config/templates/iam_member/iam_member.py"}}
  properties:
   roles:
   - role: roles/owner
     members:
     - group:foo-owner@my-domain.com`,
		},
		{
			name: "pubsub",
			configData: &testconf.ConfigData{`
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
- path: {{abs "deploy/config/templates/pubsub/pubsub.py"}}

resources:
- name: foo-topic
  type: {{abs "deploy/config/templates/pubsub/pubsub.py"}}
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
        - 'group:my-project-readwrite@my-domain.com'
      - role: roles/pubsub.viewer
        members:
        - 'group:my-project-readonly@my-domain.com'
        - 'group:another-readonly-group@googlegroups.com'
        - 'user:extra-reader@google.com'`,
		},
		{
			name: "vpc_networks",
			configData: &testconf.ConfigData{`
resources:
  vpc_networks:
  - properties:
      name: some-private
      autoCreateSubnetworks: false
      subnetworks:
      - name: some-subnetwork
        region: us-central1
        ipCidrRange: 172.16.0.0/24
        enableFlowLogs: true`},
			want: `
imports:
- path: {{abs "deploy/config/templates/network/network.py"}}

resources:
- name: some-private
  type: {{abs "deploy/config/templates/network/network.py"}}
  properties:
    name: some-private
    autoCreateSubnetworks: false
    subnetworks:
    - name: some-subnetwork
      region: us-central1
      ipCidrRange: 172.16.0.0/24
      enableFlowLogs: true`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conf, project := testconf.ConfigAndProject(t, tc.configData)

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

			if err := Apply(conf, project); err != nil {
				t.Fatalf("Deploy: %v", err)
			}

			want := []upsertCall{
				{"data-protect-toolkit-resources", wantResourceDeployment(t, tc.want), project.ID},
				{"data-protect-toolkit-audit-my-project", parseTemplateToDeployment(t, auditDeploymentYAML), project.ID},
			}

			// allow imports and resources to be in any order
			opts := []cmp.Option{
				cmpopts.SortSlices(func(a, b *deploymentmanager.Resource) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b *deploymentmanager.Import) bool { return a.Path < b.Path }),
			}
			if diff := cmp.Diff(got, want, opts...); diff != "" {
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

	cmdOutput = func(cmd *exec.Cmd) ([]byte, error) {
		args := []string{"gcloud", "logging", "sinks", "describe"}
		if cmp.Equal(cmd.Args[:len(args)], args) {
			return []byte(logSinkJSON), nil
		}
		return nil, fmt.Errorf("unexpected args: %v", cmd.Args)
	}

	_, project := testconf.ConfigAndProject(t, nil)
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
	// TODO: change this to UnmarshalStrict once
	// https://github.com/ghodss/yaml/issues/50 is fixed.
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
