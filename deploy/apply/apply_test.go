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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/deploymentmanager"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ghodss/yaml"
)

const wantPreRequisiteDeploymentYAML = `
imports:
- path: {{abs "templates/audit_log_config.py"}}
- path: {{abs "templates/chc_resource/chc_res_type_provider.jinja"}}

resources:
- name: enable-all-audit-log-policies
  type: {{abs "templates/audit_log_config.py"}}
  properties: {}
- name: chc-type-provider
  type: {{abs "templates/chc_resource/chc_res_type_provider.jinja"}}
  properties: {}
`

const wantAuditDeploymentYAML = `
imports:
- path: {{abs "config/templates/bigquery/bigquery_dataset.py"}}
- path: {{abs "config/templates/gcs_bucket/gcs_bucket.py"}}

resources:
- name: audit_logs
  type: {{abs "config/templates/bigquery/bigquery_dataset.py"}}
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
  type: {{abs "config/templates/gcs_bucket/gcs_bucket.py"}}
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
- path: {{abs "config/templates/iam_member/iam_member.py"}}

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
  type: {{abs "config/templates/iam_member/iam_member.py"}}
  properties:
    roles:
    - role: roles/owner
      members:
      - group:my-project-owners@my-domain.com
    - role: roles/iam.securityReviewer
      members:
      - group:my-project-auditors@my-domain.com
`

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

type testRunner struct {
	called []string
	// Any command called on the runner will return the following output and error unless treated differently.
	cmdOutput string
	cmdErr    error
}

func (r *testRunner) CmdRun(cmd *exec.Cmd) error {
	r.called = append(r.called, strings.Join(cmd.Args, " "))
	return r.cmdErr
}

func (r *testRunner) CmdOutput(cmd *exec.Cmd) ([]byte, error) {
	const logSinkJSON = `{
			"createTime": "2019-04-15T20:00:16.734389353Z",
			"destination": "bigquery.googleapis.com/projects/my-project/datasets/audit_logs",
			"filter": "logName:\"logs/cloudaudit.googleapis.com\"",
			"name": "audit-logs-to-bigquery",
			"outputVersionFormat": "V2",
			"updateTime": "2019-04-15T20:00:16.734389353Z",
			"writerIdentity": "serviceAccount:p12345-999999@gcp-sa-logging.iam.gserviceaccount.com"
		}`

	r.called = append(r.called, strings.Join(cmd.Args, " "))
	switch cmdStr := strings.Join(cmd.Args, " "); {
	case contains(cmdStr, "logging sinks describe audit-logs-to-bigquery", "--format json"):
		return []byte(logSinkJSON), r.cmdErr
	case contains(cmdStr, "config get-value account", "--format json"):
		return []byte(`"foo-user@my-domain.com"`), r.cmdErr
	case contains(cmdStr, "projects get-iam-policy"):
		return []byte("{}"), r.cmdErr
	default:
		return []byte(r.cmdOutput), r.cmdErr
	}
}

func (r *testRunner) CmdCombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	r.called = append(r.called, strings.Join(cmd.Args, " "))
	return []byte(r.cmdOutput), r.cmdErr
}

func TestDeploy(t *testing.T) {
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
- path: {{abs "config/templates/bigquery/bigquery_dataset.py"}}
resources:
- name: foo-dataset
  type: {{abs "config/templates/bigquery/bigquery_dataset.py"}}
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
- path: {{abs "config/templates/cloud_router/cloud_router.py"}}

resources:
- name: bar-cloud-router
  type: {{abs "config/templates/cloud_router/cloud_router.py"}}
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
- path: {{abs "config/templates/firewall/firewall.py"}}

resources:
- name: foo-firewall-rules
  type: {{abs "config/templates/firewall/firewall.py"}}
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
- path: {{abs "config/templates/instance/instance.py"}}

resources:
- name: foo-instance
  type: {{abs "config/templates/instance/instance.py"}}
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
  - properties:
      name: foo-bucket
      location: us-east1`},
			want: `
imports:
- path: {{abs "config/templates/gcs_bucket/gcs_bucket.py"}}

resources:
- name: foo-bucket
  type: {{abs "config/templates/gcs_bucket/gcs_bucket.py"}}
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
      logBucket: my-project-logs`,
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
- path: {{abs "config/templates/iam_custom_role/project_custom_role.py"}}

resources:
- name: fooCustomRole
  type:  {{abs "config/templates/iam_custom_role/project_custom_role.py"}}
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
  type:  {{abs "config/templates/iam_member/iam_member.py"}}
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
- path: {{abs "config/templates/ip_reservation/ip_address.py"}}

resources:
- name: mybarip
  type: {{abs "config/templates/ip_reservation/ip_address.py"}}
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
- path: {{abs "config/templates/pubsub/pubsub.py"}}

resources:
- name: foo-topic
  type: {{abs "config/templates/pubsub/pubsub.py"}}
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
			name: "route",
			configData: &testconf.ConfigData{`
resources:
  routes:
  - properties:
      name: foo-route
      network: foo-network
      region: us-central1
      routeType: vpntunnel
      vpnTunnelName: bar-tunnel
      priority: 20000
      destRange: "10.0.0.0/24"
      tags:
        - my-iproute-tag`},
			want: `
imports:
- path: {{abs "config/templates/route/single_route.py"}}

resources:
- name: foo-route
  type: {{abs "config/templates/route/single_route.py"}}
  properties:
      name: foo-route
      network: foo-network
      region: us-central1
      routeType: vpntunnel
      vpnTunnelName: bar-tunnel
      priority: 20000
      destRange: "10.0.0.0/24"
      tags:
        - my-iproute-tag`,
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
- path: {{abs "config/templates/network/network.py"}}

resources:
- name: some-private
  type: {{abs "config/templates/network/network.py"}}
  properties:
    name: some-private
    autoCreateSubnetworks: false
    subnetworks:
    - name: some-subnetwork
      region: us-central1
      ipCidrRange: 172.16.0.0/24
      enableFlowLogs: true`,
		},
		{
			name: "vpn",
			configData: &testconf.ConfigData{`
resources:
  vpns:
  - name: foo-vpn
    properties:
      region: us-central1
      networkURL: foo-network
      peerAddress: "33.33.33.33"
      sharedSecret: "INSERT_SECRET_HERE"
      localTrafficSelector: ["0.0.0.0/0"]
      remoteTrafficSelector: ["0.0.0.0/0"]`},
			want: `
imports:
- path: {{abs "config/templates/vpn/vpn.py"}}

resources:
- name: foo-vpn
  type: {{abs "config/templates/vpn/vpn.py"}}
  properties:
      region: us-central1
      networkURL: foo-network
      peerAddress: "33.33.33.33"
      sharedSecret: "INSERT_SECRET_HERE"
      localTrafficSelector: ["0.0.0.0/0"]
      remoteTrafficSelector: ["0.0.0.0/0"]`,
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
			upsertDeployment = func(name string, deployment *deploymentmanager.Deployment, projectID string, _ runner.Runner) error {
				got = append(got, upsertCall{name, deployment, projectID})
				return nil
			}

			if err := DeployResources(conf, project, &testRunner{}); err != nil {
				t.Fatalf("DeployResources: %v", err)
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
			wantProjectNum: "0000",
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
			wantProjectNum: "0000",
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
			wantErr: true,
		},
		{
			name:       "invalid_project_state",
			projectID:  "my_project",
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
			wantErr: true,
		},
		{
			name:       "invalid_project_folder_id",
			projectID:  "my_project",
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
			wantErr: true,
		},
		{
			name:       "invalid_project_org_id",
			projectID:  "my_project",
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
			wantErr: true,
		},
		{
			name:       "invalid_response",
			projectID:  "my_project",
			parentType: "folder",
			parentID:   "5678",
			resp:       `{}`,
			wantErr:    true,
		},
		{
			name:      "cmd_failed",
			projectID: "my_project",
			err:       errors.New("Project does not exist"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotProjectNum, gotErr := verifyProject(tc.projectID, tc.projectNum, tc.parentType, tc.parentID, &testRunner{cmdOutput: tc.resp, cmdErr: tc.err})
			if gotProjectNum != tc.wantProjectNum || (gotErr != nil) != tc.wantErr {
				t.Fatalf("verifyProject(%s, %s, %s, %s) = %q, %v; want %q, error %t", tc.projectID, tc.projectNum, tc.parentType, tc.parentID, gotProjectNum, gotErr, tc.wantProjectNum, tc.wantErr)
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &testRunner{}
			if err := setupBilling(tc.project, tc.defaultBA, r); err != nil {
				t.Fatalf("setupBilling = %v", err)
			}
			if len(r.called) > 1 {
				t.Fatal("cmdRun is called more than once")
			}
			if !strings.Contains(r.called[0], tc.wantSubstring) {
				t.Fatalf("command %q does not contain expected substring %q", r.called[0], tc.wantSubstring)
			}
		})
	}
}

func TestCreateDeletionLien(t *testing.T) {
	tests := []struct {
		name        string
		project     *config.Project
		cmdOutput   string
		wantCmdCnts int
	}{
		{
			name: "lien_not_requested",
			project: &config.Project{
				ID: "my_project",
			},
			wantCmdCnts: 0,
		},
		{
			name: "lien_requested_and_created",
			project: &config.Project{
				ID:                 "my_project",
				CreateDeletionLien: true,
			},
			wantCmdCnts: 2,
		},
		{
			name: "lien_requested_and_already_exist",
			project: &config.Project{
				ID:                 "my_project",
				CreateDeletionLien: true,
			},
			cmdOutput:   "resourcemanager.projects.delete",
			wantCmdCnts: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &testRunner{cmdOutput: tc.cmdOutput}
			if err := createDeletionLien(tc.project, r); err != nil {
				t.Fatalf("createDeletionLien = %v", err)
			}
			if len(r.called) != tc.wantCmdCnts {
				t.Fatalf("createDeletionLien(%v) cmd counts = %d; want %d", tc.project, len(r.called), tc.wantCmdCnts)
			}
		})
	}
}

func TestStackdriverAccountExist(t *testing.T) {
	projectID := "my_project"
	tests := []struct {
		name      string
		cmdError  error
		cmdOutput string
		wantExist bool
		wantErr   bool
	}{
		{
			name:      "account_exist",
			cmdOutput: "Listed 0 items.",
			wantExist: true,
		},
		{
			name:      "account_does_not_exist",
			cmdError:  errors.New(""),
			cmdOutput: fmt.Sprintf("INVALID_ARGUMENT: 'projects/%s' is not a Stackdriver workspace.", projectID),
		},
		{
			name:      "unexpected_cmd_error",
			cmdError:  errors.New(""),
			cmdOutput: fmt.Sprintf("does not have permission to access project [%s]", projectID),
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exist, err := stackdriverAccountExists(projectID, &testRunner{cmdOutput: tc.cmdOutput, cmdErr: tc.cmdError})
			if exist != tc.wantExist || (err != nil) != tc.wantErr {
				t.Fatalf("stackdriverAccountExists(%s) = %t, %v; want %t, error %t", projectID, exist, err, tc.wantExist, tc.wantErr)
			}
		})
	}
}

func TestGetLogSinkServiceAccount(t *testing.T) {
	_, project := testconf.ConfigAndProject(t, nil)
	got, err := getLogSinkServiceAccount(project, "audit-logs-to-bigquery", &testRunner{})
	want := "p12345-999999@gcp-sa-logging.iam.gserviceaccount.com"
	if got != want || err != nil {
		t.Errorf("getLogSinkServiceAccount(%v) = %q, %v; want %q, nil", project, got, err, want)
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
