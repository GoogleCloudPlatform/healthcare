// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apply

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

type applyCall struct {
	Config  map[string]interface{}
	Imports []terraform.Import
}

const baseResourcesYAML = `
provider:
- google:
    project: my-project
- google-beta:
    project: my-project
terraform:
  required_version: '>= 0.12.0'
  backend:
    gcs:
      bucket: my-project-state
      prefix: resources
data:
- google_project:
    my-project:
      project_id: my-project
resource:
- google_project_iam_audit_config:
    project:
      project: my-project
      service: allServices
      audit_log_config:
      - log_type: DATA_READ
      - log_type: DATA_WRITE
      - log_type: ADMIN_READ
- google_project_iam_member:
    project:
      for_each:
        'roles/owner group:my-project-owners@my-domain.com':
          role: roles/owner
          member: group:my-project-owners@my-domain.com
        'roles/iam.securityReviewer group:my-project-auditors@my-domain.com':
          role: roles/iam.securityReviewer
          member: group:my-project-auditors@my-domain.com
        'roles/storage.admin group:my-project-owners@my-domain.com':
          role: roles/storage.admin
          member: group:my-project-owners@my-domain.com
      project: my-project
      role: ${each.value.role}
      member: ${each.value.member}
- google_logging_metric:
    bigquery-settings-change-count:
      name: bigquery-settings-change-count
      project: my-project
      description: Count of bigquery permission changes.
      filter: resource.type="bigquery_resource" AND protoPayload.methodName="datasetservice.update"
      metric_descriptor:
        metric_kind: DELTA
        value_type: INT64
        labels:
        - key: user
          description: Unexpected user
          value_type: STRING
      label_extractors:
        user: EXTRACT(protoPayload.authenticationInfo.principalEmail)
- google_logging_metric:
    bucket-permission-change-count:
      name: bucket-permission-change-count
      project: my-project
      description: Count of GCS permissions changes.
      filter: |-
        resource.type=gcs_bucket AND protoPayload.serviceName=storage.googleapis.com AND
        (protoPayload.methodName=storage.setIamPermissions OR protoPayload.methodName=storage.objects.update)
      metric_descriptor:
        metric_kind: DELTA
        value_type: INT64
        labels:
        - key: user
          description: Unexpected user
          value_type: STRING
      label_extractors:
        user: EXTRACT(protoPayload.authenticationInfo.principalEmail)
- google_logging_metric:
    iam-policy-change-count:
      name: iam-policy-change-count
      project: my-project
      description: Count of IAM policy changes.
      filter: protoPayload.methodName="SetIamPolicy" OR protoPayload.methodName:".setIamPolicy"
      metric_descriptor:
        metric_kind: DELTA
        value_type: INT64
        labels:
        - key: user
          description: Unexpected user
          value_type: STRING
      label_extractors:
        user: EXTRACT(protoPayload.authenticationInfo.principalEmail)
`

var resourcesImports = []terraform.Import{
	{Address: "google_logging_metric.bigquery-settings-change-count", ID: "bigquery-settings-change-count"},
	{Address: "google_logging_metric.bucket-permission-change-count", ID: "bucket-permission-change-count"},
	{Address: "google_logging_metric.iam-policy-change-count", ID: "iam-policy-change-count"},
}

// TODO: Add integration test
func TestResources(t *testing.T) {
	config.EnableTerraform = true
	runner.StubFakeCmds()

	origCmdOutput := runner.CmdOutput
	defer func() {
		runner.CmdOutput = origCmdOutput
	}()

	runner.CmdOutput = func(cmd *exec.Cmd) ([]byte, error) {
		args := strings.Join(cmd.Args, " ")
		switch {
		case strings.Contains(args, "resource-manager liens list"):
			return []byte(`[{"name": "p1111-l2222", "restrictions": ["resourcemanager.projects.delete"]}]`), nil
		case strings.Contains(args, "monitoring channels list"):
			return []byte(`[{"displayName": "Email", "name": "projects/my-project/notificationChannels/111"}]`), nil
		case strings.Contains(args, "monitoring policies list"):
			// Intentionally omit "IAM Policy Change Alert" to validate it doesn't try to be imported.
			return []byte(`[
  {"displayName": "Bigquery Update Alert", "name": "projects/my-project/alertPolicies/222"},
  {"displayName": "Bucket Permission Change Alert", "name": "projects/my-project/alertPolicies/333"}
]`), nil
		default:
			return origCmdOutput(cmd)
		}
	}

	tests := []struct {
		name          string
		data          *testconf.ConfigData
		wantResources string
		wantImports   []terraform.Import
	}{
		{
			name: "no_resources",
		},
		{
			name: "bigquery_dataset",
			data: &testconf.ConfigData{`
bigquery_datasets:
- dataset_id: foo_dataset
  location: US`},
			wantResources: `
- google_bigquery_dataset:
    foo_dataset:
      dataset_id: foo_dataset
      project: my-project
      location: US
      access:
      - role: OWNER
        group_by_email: my-project-owners@my-domain.com`,
			wantImports: []terraform.Import{{
				Address: "google_bigquery_dataset.foo_dataset",
				ID:      "my-project:foo_dataset",
			}},
		},
		{
			name: "compute_firewall",
			data: &testconf.ConfigData{`
compute_firewalls:
- name: foo-firewall
  network: default
  allow:
    protocol: icmp
`},
			wantResources: `
- google_compute_firewall:
    foo-firewall:
      name: foo-firewall
      project: my-project
      network: default
      allow:
        protocol: icmp`,
			wantImports: []terraform.Import{{
				Address: "google_compute_firewall.foo-firewall",
				ID:      "my-project/foo-firewall",
			}},
		},
		{
			name: "compute_image",
			data: &testconf.ConfigData{`
compute_images:
- name: foo-image
  raw_disk:
    source: https://storage.googleapis.com/bosh-cpi-artifacts/bosh-stemcell-3262.4-google-kvm-ubuntu-trusty-go_agent-raw.tar.gz
`},
			wantResources: `
- google_compute_image:
    foo-image:
      name: foo-image
      project: my-project
      raw_disk:
        source: https://storage.googleapis.com/bosh-cpi-artifacts/bosh-stemcell-3262.4-google-kvm-ubuntu-trusty-go_agent-raw.tar.gz`,
			wantImports: []terraform.Import{{
				Address: "google_compute_image.foo-image",
				ID:      "my-project/foo-image",
			}},
		},
		{
			name: "compute_instance",
			data: &testconf.ConfigData{`
compute_instances:
- name: foo-instance
  zone: us-central1-a
  machine_type: n1-standard-1
  boot_disk:
    initialize_params:
      image: debian-cloud/debian-9
  network_interface:
    network: default
`},
			wantResources: `
- google_compute_instance:
    foo-instance:
      name: foo-instance
      project: my-project
      zone: us-central1-a
      machine_type: n1-standard-1
      boot_disk:
        initialize_params:
          image: debian-cloud/debian-9
      network_interface:
        network: default`,
			wantImports: []terraform.Import{{
				Address: "google_compute_instance.foo-instance",
				ID:      "my-project/us-central1-a/foo-instance",
			}},
		},
		{
			name: "healthcare_dataset",
			data: &testconf.ConfigData{`
healthcare_datasets:
- name: foo-dataset
  location: us-central1
  _iam_members:
  - role: roles/editor
    member: user:foo@my-domain.com
  _dicom_stores:
  - name: foo-dicom-store
    _iam_members:
    - role: roles/viewer
      member: user:bar@my-domain.com
  _fhir_stores:
  - name: foo-fhir-store
    _iam_members:
    - role: roles/viewer
      member: user:bar@my-domain.com
  _hl7_v2_stores:
  - name: foo-hl7-v2-store
    _iam_members:
    - role: roles/viewer
      member: user:bar@my-domain.com
`},
			wantResources: `
- google_healthcare_dataset:
    foo-dataset:
      name: foo-dataset
      location: us-central1
      project: my-project
      provider: google-beta
- google_healthcare_dataset_iam_member:
    foo-dataset:
      for_each:
        'roles/editor user:foo@my-domain.com':
          role: roles/editor
          member: user:foo@my-domain.com
      role: ${each.value.role}
      member: ${each.value.member}
      dataset_id: ${google_healthcare_dataset.foo-dataset.id}
      provider: google-beta
- google_healthcare_dicom_store:
    foo-dataset_foo-dicom-store:
      name: foo-dicom-store
      dataset: ${google_healthcare_dataset.foo-dataset.id}
      provider: google-beta
- google_healthcare_dicom_store_iam_member:
    foo-dataset_foo-dicom-store:
      for_each:
        'roles/viewer user:bar@my-domain.com':
          role: roles/viewer
          member: user:bar@my-domain.com
      role: ${each.value.role}
      member: ${each.value.member}
      dicom_store_id: ${google_healthcare_dicom_store.foo-dataset_foo-dicom-store.id}
      provider: google-beta
- google_healthcare_fhir_store:
    foo-dataset_foo-fhir-store:
      name: foo-fhir-store
      dataset: ${google_healthcare_dataset.foo-dataset.id}
      provider: google-beta
- google_healthcare_fhir_store_iam_member:
    foo-dataset_foo-fhir-store:
      for_each:
        'roles/viewer user:bar@my-domain.com':
          role: roles/viewer
          member: user:bar@my-domain.com
      role: ${each.value.role}
      member: ${each.value.member}
      fhir_store_id: ${google_healthcare_fhir_store.foo-dataset_foo-fhir-store.id}
      provider: google-beta
- google_healthcare_hl7_v2_store:
    foo-dataset_foo-hl7-v2-store:
      name: foo-hl7-v2-store
      dataset: ${google_healthcare_dataset.foo-dataset.id}
      provider: google-beta
- google_healthcare_hl7_v2_store_iam_member:
    foo-dataset_foo-hl7-v2-store:
      for_each:
        'roles/viewer user:bar@my-domain.com':
          role: roles/viewer
          member: user:bar@my-domain.com
      role: ${each.value.role}
      member: ${each.value.member}
      hl7_v2_store_id: ${google_healthcare_hl7_v2_store.foo-dataset_foo-hl7-v2-store.id}
      provider: google-beta`,
		},
		{
			name: "monitoring_notification_channel",
			data: &testconf.ConfigData{`
monitoring_notification_channels:
- display_name: Email
  _email: my-auditors@my-domain.com`},
			wantResources: `
- google_monitoring_alert_policy:
    bigquery_update_alert:
      display_name: Bigquery Update Alert
      project: my-project
      documentation:
        content: This policy ensures the designated user/group is notified when Bigquery dataset settings are altered.
        mime_type: text/markdown
      combiner: AND
      conditions:
      - display_name: No tolerance on bigquery-settings-change-count!
        condition_threshold:
          comparison: COMPARISON_GT
          duration: 0s
          filter: resource.type="global" AND metric.type="logging.googleapis.com/user/${google_logging_metric.bigquery-settings-change-count.name}"
      notification_channels:
      - ${google_monitoring_notification_channel.email.name}
- google_monitoring_alert_policy:
    bucket_permission_change_alert:
      display_name: Bucket Permission Change Alert
      project: my-project
      documentation:
        content: This policy ensures the designated user/group is notified when bucket/object permissions are altered.
        mime_type: text/markdown
      combiner: AND
      conditions:
      - display_name: No tolerance on bucket-permission-change-count!
        condition_threshold:
          comparison: COMPARISON_GT
          duration: 0s
          filter: resource.type="gcs_bucket" AND metric.type="logging.googleapis.com/user/${google_logging_metric.bucket-permission-change-count.name}"
      notification_channels:
      - ${google_monitoring_notification_channel.email.name}
- google_monitoring_alert_policy:
    iam_policy_change_alert:
      display_name: IAM Policy Change Alert
      project: my-project
      documentation:
        content: This policy ensures the designated user/group is notified when IAM policies are altered.
        mime_type: text/markdown
      combiner: AND
      conditions:
      - display_name: No tolerance on iam-policy-change-count!
        condition_threshold:
          comparison: COMPARISON_GT
          duration: 0s
          filter: resource.type=one_of("global","pubsub_topic","pubsub_subscription","gce_instance") AND metric.type="logging.googleapis.com/user/${google_logging_metric.iam-policy-change-count.name}"
      notification_channels:
      - ${google_monitoring_notification_channel.email.name}
- google_monitoring_notification_channel:
    email:
      display_name: Email
      project: my-project
      type: email
      labels:
        email_address: my-auditors@my-domain.com`,
			wantImports: []terraform.Import{
				{Address: "google_monitoring_alert_policy.bigquery_update_alert", ID: "projects/my-project/alertPolicies/222"},
				{Address: "google_monitoring_alert_policy.bucket_permission_change_alert", ID: "projects/my-project/alertPolicies/333"},
				{Address: "google_monitoring_notification_channel.email", ID: "projects/my-project/notificationChannels/111"},
			},
		},
		{
			name: "storage_bucket",
			data: &testconf.ConfigData{`
storage_buckets:
- name: foo-bucket
  location: US
  _iam_members:
  - role: roles/storage.admin
    member: user:foo-user@my-domain.com`},
			wantResources: `
- google_storage_bucket:
    foo-bucket:
      name: foo-bucket
      project: my-project
      location: US
      versioning:
        enabled: true
- google_storage_bucket_iam_member:
    foo-bucket:
      for_each:
        'roles/storage.admin user:foo-user@my-domain.com':
          role: roles/storage.admin
          member: user:foo-user@my-domain.com
      bucket: '${google_storage_bucket.foo-bucket.name}'
      role: '${each.value.role}'
      member: '${each.value.member}'`,
			wantImports: []terraform.Import{{
				Address: "google_storage_bucket.foo-bucket",
				ID:      "my-project/foo-bucket",
			}},
		},
		{
			name: "project_iam_custom_role",
			data: &testconf.ConfigData{`
project_iam_custom_roles:
- role_id: myCustomRole
  title: "My Custom Role"
  description: "A description"
  permissions:
  - iam.roles.list
  - iam.roles.create
  - iam.roles.delete`},
			wantResources: `
- google_project_iam_custom_role:
    myCustomRole:
      role_id: myCustomRole
      project: my-project
      title: "My Custom Role"
      description: "A description"
      permissions:
      - iam.roles.list
      - iam.roles.create
      - iam.roles.delete`,
			wantImports: []terraform.Import{{
				Address: "google_project_iam_custom_role.myCustomRole",
				ID:      "projects/my-project/roles/myCustomRole",
			}},
		},
		{
			name: "pubsub",
			data: &testconf.ConfigData{`
pubsub_topics:
- name: foo-topic
  _subscriptions:
  - name: foo-subscription
    message_retention_duration: 600s
    retain_acked_messages: true
    ack_deadline_seconds: 20
    _iam_members:
    - role: roles/editor
      member: user:foo-user@my-domain.com`},
			wantResources: `
- google_pubsub_topic:
    foo-topic:
      name: foo-topic
      project: my-project
- google_pubsub_subscription:
    foo-subscription:
      name: foo-subscription
      project: my-project
      topic: ${google_pubsub_topic.foo-topic.name}
      message_retention_duration: 600s
      retain_acked_messages: true
      ack_deadline_seconds: 20
- google_pubsub_subscription_iam_member:
    foo-subscription:
      for_each:
        'roles/editor user:foo-user@my-domain.com':
          role: roles/editor
          member: user:foo-user@my-domain.com
      subscription: ${google_pubsub_subscription.foo-subscription.name}
      role: ${each.value.role}
      member: ${each.value.member}`,
			wantImports: []terraform.Import{
				{Address: "google_pubsub_topic.foo-topic", ID: "my-project/foo-topic"},
				{Address: "google_pubsub_subscription.foo-subscription", ID: "my-project/foo-subscription"},
			},
		},
		{
			name: "resource_manager_lien",
			data: &testconf.ConfigData{`
resource_manager_liens:
- _project_deletion: true
- reason: Custom reason
  origin: custom-origin
  parent: projects/my-project
  restrictions:
  - foo.delete`},
			wantResources: `
- google_resource_manager_lien:
    managed_project_deletion_lien:
      reason: Managed project deletion lien
      origin: managed-terraform
      parent: projects/my-project
      restrictions:
      - resourcemanager.projects.delete
- google_resource_manager_lien:
    custom_reason:
      reason: Custom reason
      origin: custom-origin
      parent: projects/my-project
      restrictions:
      - foo.delete`,
			wantImports: []terraform.Import{{
				Address: "google_resource_manager_lien.managed_project_deletion_lien", ID: "my-project/p1111-l2222",
			}},
		},
		{
			name: "service_account",
			data: &testconf.ConfigData{`
service_accounts:
- account_id: foo-account
  display_name: Foo account`},
			wantResources: `
- google_service_account:
    foo-account:
      account_id: foo-account
      project: my-project
      display_name: Foo account`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, project := testconf.ConfigAndProject(t, tc.data)

			var got []applyCall
			terraformApply = func(config *terraform.Config, _ string, opts *terraform.Options) error {
				got = append(got, makeApplyCall(t, config, opts))
				return nil
			}

			if err := resources(project, &Options{ImportExisting: true}); err != nil {
				t.Fatalf("deployTerraform: %v", err)
			}

			want := []applyCall{{
				Config:  unmarshal(t, baseResourcesYAML+tc.wantResources),
				Imports: append(resourcesImports, tc.wantImports...),
			}}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("terraform calls differ (-got, +want):\n%v", diff)
			}
		})
	}
}

func TestCreateProject(t *testing.T) {
	config, project := testconf.ConfigAndProject(t, nil)

	var got []applyCall
	terraformApply = func(config *terraform.Config, _ string, opts *terraform.Options) error {
		got = append(got, makeApplyCall(t, config, opts))
		return nil
	}

	want := []applyCall{{
		Config: unmarshal(t, `
terraform:
  required_version: ">= 0.12.0"

resource:
- google_project:
    project:
      project_id: my-project
      name: my-project
      folder_id: '98765321'
      billing_account: 000000-000000-000000
- google_storage_bucket:
    my-project-state:
      name: my-project-state
      project: my-project
      location: US
      versioning:
        enabled: true
      depends_on:
      - google_project.project`),
		Imports: []terraform.Import{
			{Address: "google_project.project", ID: "my-project"},
			{Address: "google_storage_bucket.my-project-state", ID: "my-project/my-project-state"},
		},
	}}

	if err := createProjectTerraform(config, project); err != nil {
		t.Fatalf("createProjectTerraform: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("terraform calls differ (-got, +want):\n%v", diff)
	}
}

func TestAuditResources(t *testing.T) {
	config, project := testconf.ConfigAndProject(t, nil)

	var got []applyCall
	terraformApply = func(config *terraform.Config, _ string, opts *terraform.Options) error {
		got = append(got, makeApplyCall(t, config, opts))
		return nil
	}

	want := []applyCall{{
		Config: unmarshal(t, `
terraform:
  required_version: '>= 0.12.0'
  backend:
    gcs:
      bucket: my-project-state
      prefix: audit-my-project
resource:
- google_logging_project_sink:
    audit-logs-to-bigquery:
      name: audit-logs-to-bigquery
      project: my-project
      filter: 'logName:"logs/cloudaudit.googleapis.com"'
      destination: bigquery.googleapis.com/projects/my-project/datasets/audit_logs
      unique_writer_identity: true
- google_bigquery_dataset:
    audit_logs:
      dataset_id: audit_logs
      project: my-project
      location: US
      access:
      - role: OWNER
        group_by_email: my-project-owners@my-domain.com
      - role: READER
        group_by_email: my-project-auditors@my-domain.com
      - role: WRITER
        user_by_email: '${replace(google_logging_project_sink.audit-logs-to-bigquery.writer_identity, "serviceAccount:", "")}'
- google_storage_bucket:
    my-project-logs:
      name: my-project-logs
      project: my-project
      location: US
      storage_class: MULTI_REGIONAL
      versioning:
        enabled: true
- google_storage_bucket_iam_member:
    my-project-logs:
      for_each:
        'roles/storage.admin group:my-project-owners@my-domain.com':
          role: roles/storage.admin
          member: group:my-project-owners@my-domain.com
        'roles/storage.objectCreator group:cloud-storage-analytics@google.com':
          role: roles/storage.objectCreator
          member: group:cloud-storage-analytics@google.com
        'roles/storage.objectViewer group:my-project-auditors@my-domain.com':
          role: roles/storage.objectViewer
          member: group:my-project-auditors@my-domain.com
      bucket: ${google_storage_bucket.my-project-logs.name}
      role: ${each.value.role}
      member: ${each.value.member}
`),
		Imports: []terraform.Import{
			{Address: "google_logging_project_sink.audit-logs-to-bigquery", ID: "projects/my-project/sinks/audit-logs-to-bigquery"},
			{Address: "google_bigquery_dataset.audit_logs", ID: "my-project:audit_logs"},
			{Address: "google_storage_bucket.my-project-logs", ID: "my-project/my-project-logs"},
		},
	}}

	if err := auditResources(config, project); err != nil {
		t.Fatalf("auditResources: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("terraform calls differ (-got, +want):\n%v", diff)
	}
}

func TestServices(t *testing.T) {
	_, project := testconf.ConfigAndProject(t, nil)

	var got []applyCall
	terraformApply = func(config *terraform.Config, _ string, opts *terraform.Options) error {
		got = append(got, makeApplyCall(t, config, opts))
		return nil
	}

	want := []applyCall{{
		Config: unmarshal(t, `
terraform:
  required_version: '>= 0.12.0'
  backend:
    gcs:
      bucket: my-project-state
      prefix: services
resource:
- google_project_service:
    project:
      for_each:
        'bigquery-json.googleapis.com': true
        'bigquerystorage.googleapis.com': true
        'cloudresourcemanager.googleapis.com': true
        'logging.googleapis.com': true
      project: my-project
      service: ${each.key}`),
	}}

	if err := services(project); err != nil {
		t.Fatalf("services: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("terraform calls differ (-got, +want):\n%v", diff)
	}
}

func makeApplyCall(t *testing.T, config *terraform.Config, opts *terraform.Options) applyCall {
	b, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("json.Marshal(%s) = %v", config, err)
	}

	call := applyCall{
		Config: unmarshal(t, string(b)),
	}
	if opts != nil {
		call.Imports = opts.Imports
	}
	return call
}

// unmarshal is a helper to unmarshal a yaml or json string to an interface (map).
// Note: the actual configs will always be json, but we allow yaml in tests to make them easier to write in test cases.
func unmarshal(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	out := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}
	return out
}
