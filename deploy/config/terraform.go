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

package config

import (
	"errors"
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
)

func (p *Project) initTerraform(auditProject *Project) error {
	if err := p.initTerraformAuditResources(auditProject); err != nil {
		return fmt.Errorf("failed to init audit resources: %v", err)
	}
	if err := p.initServices(); err != nil {
		return fmt.Errorf("failed to init services: %v", err)
	}

	p.addDefaultIAM()
	p.addDefaultMonitoring()

	// At least have one owner access set to override default accesses
	// (https://cloud.google.com/bigquery/docs/reference/rest/v2/datasets).
	for _, d := range p.BigqueryDatasets {
		d.Accesses = append(d.Accesses, &tfconfig.Access{Role: "OWNER", GroupByEmail: p.OwnersGroup})
	}
	if p.Audit.LogsStorageBucket != nil {
		for _, b := range p.StorageBuckets {
			b.Logging = &tfconfig.Logging{LogBucket: p.Audit.LogsStorageBucket.Name}
		}
	}

	for _, r := range p.TerraformResources() {
		if err := r.Init(p.ID); err != nil {
			return fmt.Errorf("failed to init %q (%v): %v", r.ResourceType(), r, err)
		}
	}
	return nil
}

func (p *Project) initTerraformAuditResources(auditProject *Project) error {
	d := p.Audit.LogsBigqueryDataset
	if d == nil {
		return errors.New("audit.logs_bigquery_dataset must be set")
	}

	if err := d.Init(auditProject.ID); err != nil {
		return fmt.Errorf("failed to init logs bq dataset: %v", err)
	}

	d.Accesses = append(d.Accesses,
		&tfconfig.Access{Role: "OWNER", GroupByEmail: auditProject.OwnersGroup},
		&tfconfig.Access{Role: "READER", GroupByEmail: p.AuditorsGroup},
	)

	p.BQLogSinkTF = &tfconfig.LoggingSink{
		Name:                 "audit-logs-to-bigquery",
		Destination:          fmt.Sprintf("bigquery.googleapis.com/projects/%s/datasets/%s", auditProject.ID, d.DatasetID),
		Filter:               `logName:"logs/cloudaudit.googleapis.com"`,
		UniqueWriterIdentity: true,
	}
	if err := p.BQLogSinkTF.Init(p.ID); err != nil {
		return fmt.Errorf("failed to init bigquery log sink: %v", err)
	}

	b := p.Audit.LogsStorageBucket
	if b == nil {
		return nil
	}

	if err := b.Init(auditProject.ID); err != nil {
		return fmt.Errorf("failed to init logs gcs bucket: %v", err)
	}

	b.IAMMembers = append(b.IAMMembers,
		&tfconfig.StorageIAMMember{Role: "roles/storage.admin", Member: "group:" + auditProject.OwnersGroup},
		&tfconfig.StorageIAMMember{Role: "roles/storage.objectCreator", Member: accessLogsWriter},
		&tfconfig.StorageIAMMember{Role: "roles/storage.objectViewer", Member: "group:" + p.AuditorsGroup})
	return nil
}

func (p *Project) initServices() error {
	if p.Services == nil {
		p.Services = new(tfconfig.ProjectServices)
	}

	svcs := []string{
		"bigquery-json.googleapis.com", // For bigquery audit logs and datasets.
		"bigquerystorage.googleapis.com",
		"cloudresourcemanager.googleapis.com", // For project level iam policy updates.
		"logging.googleapis.com",              // For default logging metrics.
	}
	if len(p.ComputeInstances) > 0 || len(p.ComputeImages) > 0 {
		svcs = append(svcs, "compute.googleapis.com")
	}
	if len(p.HealthcareDatasets) > 0 {
		svcs = append(svcs, "healthcare.googleapis.com")
	}
	if len(p.NotificationChannels) > 0 {
		svcs = append(svcs, "monitoring.googleapis.com")
	}
	if len(p.PubsubTopics) > 0 {
		svcs = append(svcs, "pubsub.googleapis.com")
	}

	for _, svc := range svcs {
		p.Services.Services = append(p.Services.Services, &tfconfig.ProjectService{Service: svc})
	}
	// Note: services will be de-duplicated when being marshalled.
	if err := p.Services.Init(p.ID); err != nil {
		return fmt.Errorf("failed to init services: %v", err)
	}
	return nil
}

func (p *Project) addDefaultIAM() {
	// Enable all possible audit log collection.
	p.IAMAuditConfig = &tfconfig.ProjectIAMAuditConfig{
		Service: "allServices",
		AuditLogConfigs: []*tfconfig.AuditLogConfig{
			{LogType: "DATA_READ"},
			{LogType: "DATA_WRITE"},
			{LogType: "ADMIN_READ"},
		},
	}
	if p.IAMMembers == nil {
		p.IAMMembers = new(tfconfig.ProjectIAMMembers)
	}

	p.IAMMembers.Members = append(p.IAMMembers.Members,
		&tfconfig.ProjectIAMMember{Role: "roles/owner", Member: "group:" + p.OwnersGroup},
		&tfconfig.ProjectIAMMember{Role: "roles/iam.securityReviewer", Member: "group:" + p.AuditorsGroup},
	)
	if p.Audit.LogsStorageBucket != nil || len(p.StorageBuckets) > 0 {
		// roles/owner does not grant storage.buckets.setIamPolicy, so we need to add storage admin role on the owners group.
		p.IAMMembers.Members = append(p.IAMMembers.Members, &tfconfig.ProjectIAMMember{
			Role: "roles/storage.admin", Member: "group:" + p.OwnersGroup,
		})
	}
}

func (p *Project) addDefaultMonitoring() {
	type metricAndAlert struct {
		metric *tfconfig.LoggingMetric
		alert  *tfconfig.MonitoringAlertPolicy
	}

	metricAndAlerts := []metricAndAlert{
		{
			metric: &tfconfig.LoggingMetric{
				Name:        "bigquery-settings-change-count",
				Description: "Count of bigquery permission changes.",
				Filter:      `resource.type="bigquery_resource" AND protoPayload.methodName="datasetservice.update"`,
			},
			alert: &tfconfig.MonitoringAlertPolicy{
				DisplayName: "Bigquery Update Alert",
				Documentation: &tfconfig.Documentation{
					Content: "This policy ensures the designated user/group is notified when Bigquery dataset settings are altered.",
				},
				Conditions: []*tfconfig.Condition{{
					DisplayName: "No tolerance on bigquery-settings-change-count!",
					ConditionThreshold: &tfconfig.ConditionThreshold{
						Filter: `resource.type="global" AND metric.type="logging.googleapis.com/user/${google_logging_metric.bigquery-settings-change-count.name}"`,
					},
				}},
			},
		},
		{
			metric: &tfconfig.LoggingMetric{
				Name:        "bucket-permission-change-count",
				Description: "Count of GCS permissions changes.",
				Filter: `resource.type=gcs_bucket AND protoPayload.serviceName=storage.googleapis.com AND
(protoPayload.methodName=storage.setIamPermissions OR protoPayload.methodName=storage.objects.update)`,
			},
			alert: &tfconfig.MonitoringAlertPolicy{
				DisplayName: "Bucket Permission Change Alert",
				Documentation: &tfconfig.Documentation{
					Content: "This policy ensures the designated user/group is notified when bucket/object permissions are altered.",
				},
				Conditions: []*tfconfig.Condition{{
					DisplayName: "No tolerance on bucket-permission-change-count!",
					ConditionThreshold: &tfconfig.ConditionThreshold{
						Filter: `resource.type="gcs_bucket" AND metric.type="logging.googleapis.com/user/${google_logging_metric.bucket-permission-change-count.name}"`,
					},
				}},
			},
		},
		{
			metric: &tfconfig.LoggingMetric{
				Name:        "iam-policy-change-count",
				Description: "Count of IAM policy changes.",
				Filter:      `protoPayload.methodName="SetIamPolicy" OR protoPayload.methodName:".setIamPolicy"`,
			},
			alert: &tfconfig.MonitoringAlertPolicy{
				DisplayName: "IAM Policy Change Alert",
				Documentation: &tfconfig.Documentation{
					Content: "This policy ensures the designated user/group is notified when IAM policies are altered.",
				},
				Conditions: []*tfconfig.Condition{{
					DisplayName: "No tolerance on iam-policy-change-count!",
					ConditionThreshold: &tfconfig.ConditionThreshold{
						Filter: `resource.type=one_of("global","pubsub_topic","pubsub_subscription","gce_instance") AND metric.type="logging.googleapis.com/user/${google_logging_metric.iam-policy-change-count.name}"`,
					},
				}},
			},
		},
	}

	for _, ma := range metricAndAlerts {
		ma.metric.MetricDescriptor = &tfconfig.MetricDescriptor{
			MetricKind: "DELTA",
			ValueType:  "INT64",
			Labels: []*tfconfig.Label{{
				Key:         "user",
				ValueType:   "STRING",
				Description: "Unexpected user",
			}},
		}
		ma.metric.LabelExtractors = map[string]string{
			"user": "EXTRACT(protoPayload.authenticationInfo.principalEmail)",
		}

		ma.alert.Documentation.MimeType = "text/markdown"
		ma.alert.Combiner = "AND"
		for _, c := range ma.alert.Conditions {
			c.ConditionThreshold.Comparison = "COMPARISON_GT"
			c.ConditionThreshold.Duration = "0s"
		}

		p.DefaultLoggingMetrics = append(p.DefaultLoggingMetrics, ma.metric)

		if len(p.NotificationChannels) > 0 {
			for _, c := range p.NotificationChannels {
				ref := fmt.Sprintf("${%s.%s.name}", c.ResourceType(), c.ID())
				ma.alert.NotificationChannels = append(ma.alert.NotificationChannels, ref)
			}
			p.DefaultAlertPolicies = append(p.DefaultAlertPolicies, ma.alert)
		}
	}
}

// TerraformResources gets all terraform resources in this project.
func (p *Project) TerraformResources() []tfconfig.Resource {
	var rs []tfconfig.Resource
	// Put default resources first to make it easier to write tests.
	if p.IAMAuditConfig != nil {
		rs = append(rs, p.IAMAuditConfig)
	}
	if p.IAMMembers != nil {
		rs = append(rs, p.IAMMembers)
	}
	for _, r := range p.DefaultLoggingMetrics {
		rs = append(rs, r)
	}
	for _, r := range p.DefaultAlertPolicies {
		rs = append(rs, r)
	}

	for _, r := range p.BigqueryDatasets {
		rs = append(rs, r)
	}
	for _, r := range p.CloudBuildTriggers {
		rs = append(rs, r)
	}
	for _, r := range p.ComputeFirewalls {
		rs = append(rs, r)
	}
	for _, r := range p.ComputeImages {
		rs = append(rs, r)
	}
	for _, r := range p.ComputeInstances {
		rs = append(rs, r)
	}
	for _, r := range p.DataFusionInstances {
		rs = append(rs, r)
	}
	for _, r := range p.HealthcareDatasets {
		rs = append(rs, r)
	}
	for _, r := range p.IAMCustomRoles {
		rs = append(rs, r)
	}
	for _, r := range p.NotificationChannels {
		rs = append(rs, r)
	}
	for _, r := range p.PubsubTopics {
		rs = append(rs, r)
	}
	for _, r := range p.ResourceManagerLiens {
		rs = append(rs, r)
	}
	for _, r := range p.ServiceAccounts {
		rs = append(rs, r)
	}
	for _, r := range p.StorageBuckets {
		rs = append(rs, r)
	}
	return rs
}
