/*
 * Copyright 2019 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package apply

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/deploymentmanager"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ghodss/yaml"
)

const wantPreRequisiteDeploymentYAML = `
imports:
- path: {{abs "deploy/templates/audit_log_config.py"}}
- path: {{abs "deploy/templates/chc_resource/chc_res_type_provider.jinja"}}

resources:
- name: enable-all-audit-log-policies
  type: {{abs "deploy/templates/audit_log_config.py"}}
  properties: {}
- name: chc-type-provider
  type: {{abs "deploy/templates/chc_resource/chc_res_type_provider.jinja"}}
  properties: {}
`

const wantAuditDeploymentYAML = `
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
    - userByEmail: p12345-999999@gcp-sa-logging.iam.gserviceaccount.com
      role: WRITER
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
- path: {{abs "deploy/config/templates/iam_member/iam_member.py"}}

resources:
- name: audit-logs-to-bigquery
  type: logging.v2.sink
  properties:
    sink: audit-logs-to-bigquery
    destination: bigquery.googleapis.com/projects/my-project/datasets/audit_logs
    filter: 'logName:"logs/cloudaudit.googleapis.com"'
    uniqueWriterIdentity: true
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
	origCmdOutput := cmdOutput
	cmdOutput = func(cmd *exec.Cmd) ([]byte, error) {
		args := strings.Join(cmd.Args, " ")
		var res string
		switch {
		case strings.HasPrefix(args, "gcloud config get-value account"):
			res = `"foo-user@my-domain.com"`
		case strings.HasPrefix(args, "gcloud projects get-iam-policy"):
			res = "{}"
		default:
			return origCmdOutput(cmd)
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
      role: READER`,
		},
		{
			name: "cloud_router",
			configData: &testconf.ConfigData{`
resources:
  cloud_routers:
  - properties:
      name: bar-cloud-router
      network: default
      region: us-central1
      asn: 65002`},
			want: `
imports:
- path: {{abs "deploy/config/templates/cloud_router/cloud_router.py"}}

resources:
- name: bar-cloud-router
  type: {{abs "deploy/config/templates/cloud_router/cloud_router.py"}}
  properties:
      name: bar-cloud-router
      network: default
      region: us-central1
      asn: 65002`,
		},
		{
			name: "gce_firewall",
			configData: &testconf.ConfigData{`
resources:
  gce_firewalls:
  - name: foo-firewall-rules
    properties:
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
			name: "ip_address",
			configData: &testconf.ConfigData{`
resources:
  ip_addresses:
  - properties:
      name: mybarip
      region: us-central1
      ipType: REGIONAL
      description: 'my bar ip'`},
			want: `
imports:
- path: {{abs "deploy/config/templates/ip_reservation/ip_address.py"}}

resources:
- name: mybarip
  type: {{abs "deploy/config/templates/ip_reservation/ip_address.py"}}
  properties:
    name: mybarip
    region: us-central1
    ipType: REGIONAL
    description: 'my bar ip'`,
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
			name: "service_accounts",
			configData: &testconf.ConfigData{`
resources:
  service_accounts:
  - properties:
      accountId: some-service-account
      displayName: somesa`},
			want: `

resources:
- name: some-service-account
  type: iam.v1.serviceAccount
  properties:
    accountId: some-service-account
    displayName: somesa`,
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

			if err := DeployResources(conf, project, &Options{EnableTerraform: false}); err != nil {
				t.Fatalf("Deploy: %v", err)
			}

			want := []upsertCall{
				{"data-protect-toolkit-prerequisites", parseTemplateToDeployment(t, wantPreRequisiteDeploymentYAML), project.ID},
				{"data-protect-toolkit-resources", wantResourceDeployment(t, tc.want), project.ID},
				{"data-protect-toolkit-audit-my-project", parseTemplateToDeployment(t, wantAuditDeploymentYAML), project.ID},
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

func TestVerifyProject(t *testing.T) {

	tests := []struct {
		name           string
		projectID      string
		projectNum     string
		parentType     string
		parentID       string
		resp           string
		err            error
		wantProjectNum string
		wantErr        bool
	}{
		{
			name:       "valid_project",
			projectID:  "my_project",
			projectNum: "",
			parentType: "folder",
			parentID:   "5678",
			resp: `{
  "lifecycleState": "ACTIVE",
  "name": "my_project",
  "parent": {
    "id": "5678",
    "type": "folder"
  },
  "projectNumber": "0000"
}`,
			err:            nil,
			wantProjectNum: "0000",
			wantErr:        false,
		},
		{
			name:       "valid_project_with_existing_project_number",
			projectID:  "my_project",
			projectNum: "0000",
			parentType: "folder",
			parentID:   "5678",
			resp: `{
  "lifecycleState": "ACTIVE",
  "name": "my_project",
  "parent": {
    "id": "5678",
    "type": "folder"
  },
  "projectNumber": "0000"
}`,
			err:            nil,
			wantProjectNum: "0000",
			wantErr:        false,
		},
		{
			name:       "invalid_project_number",
			projectID:  "my_project",
			projectNum: "wrong_number",
			parentType: "folder",
			parentID:   "5678",
			resp: `{
  "lifecycleState": "ACTIVE",
  "name": "my_project",
  "parent": {
    "id": "5678",
    "type": "folder"
  },
  "projectNumber": "0000"
}`,
			err:            nil,
			wantProjectNum: "",
			wantErr:        true,
		},
		{
			name:       "invalid_project_state",
			projectID:  "my_project",
			projectNum: "",
			parentType: "folder",
			parentID:   "5678",
			resp: `{
  "lifecycleState": "DELETE_REQUESTED",
  "name": "my_project",
  "parent": {
    "id": "5678",
    "type": "folder"
  },
  "projectNumber": "0000"
}`,
			err:            nil,
			wantProjectNum: "",
			wantErr:        true,
		},
		{
			name:       "invalid_project_folder_id",
			projectID:  "my_project",
			projectNum: "",
			parentType: "folder",
			parentID:   "5678",
			resp: `{
  "lifecycleState": "ACTIVE",
  "name": "my_project",
  "parent": {
    "id": "wrong_id",
    "type": "folder"
  },
  "projectNumber": "0000"
}`,
			err:            nil,
			wantProjectNum: "",
			wantErr:        true,
		},
		{
			name:       "invalid_project_org_id",
			projectID:  "my_project",
			projectNum: "",
			parentType: "organization",
			parentID:   "5678",
			resp: `{
  "lifecycleState": "ACTIVE",
  "name": "my_project",
  "parent": {
    "id": "wrong_id",
    "type": "organization"
  },
  "projectNumber": "0000"
}`,
			err:            nil,
			wantProjectNum: "",
			wantErr:        true,
		},
		{
			name:           "invalid_response",
			projectID:      "my_project",
			projectNum:     "",
			parentType:     "folder",
			parentID:       "5678",
			resp:           `{}`,
			err:            nil,
			wantProjectNum: "",
			wantErr:        true,
		},
		{
			name:           "cmd_failed",
			projectID:      "my_project",
			projectNum:     "",
			parentType:     "",
			parentID:       "",
			resp:           ``,
			err:            errors.New("Project does not exist"),
			wantProjectNum: "",
			wantErr:        false,
		},
	}

	origCmdOutput := cmdOutput
	defer func() { cmdOutput = origCmdOutput }()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmdOutput = func(cmd *exec.Cmd) ([]byte, error) {
				return []byte(tc.resp), tc.err
			}
			gotProjectNum, gotErr := verifyProject(tc.projectID, tc.projectNum, tc.parentType, tc.parentID)
			if (gotErr != nil) != tc.wantErr {
				t.Fatalf("error differs: want %t, got %t", tc.wantErr, (gotErr != nil))
			}
			if gotProjectNum != tc.wantProjectNum {
				t.Fatalf("project number differs: want %q, got %q", tc.wantProjectNum, gotProjectNum)
			}
		})
	}
}

func TestSetupBilling(t *testing.T) {
	tests := []struct {
		name          string
		project       *config.Project
		defaultBA     string
		wantSubstring string
	}{
		{
			name: "project_with_billing_account",
			project: &config.Project{
				ID:             "my_project",
				BillingAccount: "my_billing_account",
			},
			defaultBA:     "default_billing_account",
			wantSubstring: "--billing-account my_billing_account",
		},
		{
			name: "project_without_billing_account",
			project: &config.Project{
				ID: "my_project",
			},
			defaultBA:     "default_billing_account",
			wantSubstring: "--billing-account default_billing_account",
		},
	}
	origCmdRun := cmdRun
	defer func() { cmdRun = origCmdRun }()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ""
			cmdRun = func(cmd *exec.Cmd) error {
				if got != "" {
					t.Fatal("cmdRun is called more than once")
				}
				got = strings.Join(cmd.Args, " ")
				return nil
			}
			if err := setupBilling(tc.project, tc.defaultBA); err != nil {
				t.Fatalf("setupBilling = %v", err)
			}
			if !strings.Contains(got, tc.wantSubstring) {
				t.Fatalf("command %q does not contain expected substring %q", got, tc.wantSubstring)
			}
		})
	}
}

func TestCreateDeletionLien(t *testing.T) {
	tests := []struct {
		name            string
		project         *config.Project
		cmdOutput       string
		wantEmptyCmdRun bool
	}{
		{
			name: "lien_not_requested",
			project: &config.Project{
				ID:                 "my_project",
				CreateDeletionLien: false,
			},
			cmdOutput:       "",
			wantEmptyCmdRun: true,
		},
		{
			name: "lien_requested_and_created",
			project: &config.Project{
				ID:                 "my_project",
				CreateDeletionLien: true,
			},
			cmdOutput:       "",
			wantEmptyCmdRun: false,
		},
		{
			name: "lien_requested_and_already_exist",
			project: &config.Project{
				ID:                 "my_project",
				CreateDeletionLien: true,
			},
			cmdOutput:       "resourcemanager.projects.delete",
			wantEmptyCmdRun: true,
		},
	}
	origCmdRun := cmdRun
	defer func() { cmdRun = origCmdRun }()
	origCmdOutput := cmdOutput
	defer func() { cmdOutput = origCmdOutput }()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			cmdRun = func(cmd *exec.Cmd) error {
				if got != "" {
					t.Fatal("cmdRun is called more than once")
				}
				got = strings.Join(cmd.Args, " ")
				return nil
			}
			cmdOutput = func(cmd *exec.Cmd) ([]byte, error) {
				return []byte(tc.cmdOutput), nil
			}
			if err := createDeletionLien(tc.project); err != nil {
				t.Fatalf("createDeletionLien = %v", err)
			}
			if (len(got) == 0) != tc.wantEmptyCmdRun {
				t.Fatalf("deletion lien creation cmdRun args differ, want empty is %t, got %t", tc.wantEmptyCmdRun, len(got) == 0)
			}
		})
	}
}

func TestGetLogSinkServiceAccount(t *testing.T) {
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

func TestMain(m *testing.M) {
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

	os.Exit(m.Run())
}
