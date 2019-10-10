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

// Package config provides utilities to parse and create project and resource configurations.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
)

// EnableTerraform determines whether terraform will be enabled or not.
// Note: The terraform state bucket does not respect this var as it is required currently for Forseti projects.
// TODO: remove this once DM has been deprecated.
var EnableTerraform = false

// accessLogsWriter is the access logs writer.
// https://cloud.google.com/storage/docs/access-logs#delivery.
const accessLogsWriter = "group:cloud-storage-analytics@google.com"

// Logging Metric names used to create logs-based-metrics and Stackdriver alerts.
const (
	IAMChangeMetricName                = "iam-policy-change-count"
	BucketPermissionChangeMetricName   = "bucket-permission-change-count"
	BQSettingChangeMetricName          = "bigquery-settings-change-count"
	BucketUnexpectedAccessMetricPrefix = "unexpected-access-"
)

// Config represents a (partial) representation of a projects YAML file.
// Only the required fields are present. See project_config.yaml.schema for details.
type Config struct {
	Overall struct {
		BillingAccount string   `json:"billing_account"`
		Domain         string   `json:"domain"`
		OrganizationID string   `json:"organization_id"`
		FolderID       string   `json:"folder_id"`
		AllowedAPIs    []string `json:"allowed_apis"`
	} `json:"overall"`

	Devops *struct {
		Project *Project `json:"project"`
	} `json:"devops"`
	AuditLogsProject    *Project   `json:"audit_logs_project"`
	Forseti             *Forseti   `json:"forseti"`
	Projects            []*Project `json:"projects"`
	GeneratedFieldsPath string     `json:"generated_fields_path"`

	// Set by helper and not directly through user defined config.
	AllGeneratedFields *AllGeneratedFields `json:"-"`
}

// Project defines a single project's configuration.
type Project struct {
	ID                  string   `json:"project_id"`
	BillingAccount      string   `json:"billing_account"`
	FolderID            string   `json:"folder_id"`
	OwnersGroup         string   `json:"owners_group"`
	AuditorsGroup       string   `json:"auditors_group"`
	DataReadWriteGroups []string `json:"data_readwrite_groups"`
	DataReadOnlyGroups  []string `json:"data_readonly_groups"`

	TerraformConfig struct {
		StateBucket *tfconfig.StorageBucket `json:"state_storage_bucket"`
	} `json:"terraform"`

	CreateDeletionLien    bool                `json:"create_deletion_lien"`
	EnabledAPIs           []string            `json:"enabled_apis"`
	ViolationExceptions   map[string][]string `json:"violation_exceptions"`
	StackdriverAlertEmail string              `json:"stackdriver_alert_email"`

	Resources struct {
		// Deployment manager resources
		BQDatasets      []*BigqueryDataset `json:"bq_datasets"`
		CHCDatasets     []*CHCDataset      `json:"chc_datasets"`
		CloudRouter     []*DefaultResource `json:"cloud_routers"`
		GCEFirewalls    []*DefaultResource `json:"gce_firewalls"`
		GCEInstances    []*GCEInstance     `json:"gce_instances"`
		GCSBuckets      []*GCSBucket       `json:"gcs_buckets"`
		GKEClusters     []*GKECluster      `json:"gke_clusters"`
		IAMCustomRoles  []*IAMCustomRole   `json:"iam_custom_roles"`
		IAMPolicies     []*IAMPolicy       `json:"iam_policies"`
		IPAddresses     []*DefaultResource `json:"ip_addresses"`
		Pubsubs         []*Pubsub          `json:"pubsubs"`
		Routes          []*DefaultResource `json:"routes"`
		ServiceAccounts []*ServiceAccount  `json:"service_accounts"`
		VPCNetworks     []*DefaultResource `json:"vpc_networks"`
		VPNs            []*DefaultResource `json:"vpns"`

		// Kubectl resources
		GKEWorkloads []*GKEWorkload `json:"gke_workloads"`
	} `json:"resources"`

	// Terraform resources
	BigqueryDatasets     []*tfconfig.BigqueryDataset               `json:"bigquery_datasets"`
	ComputeFirewalls     []*tfconfig.ComputeFirewall               `json:"compute_firewalls"`
	ComputeImages        []*tfconfig.ComputeImage                  `json:"compute_images"`
	ComputeInstances     []*tfconfig.ComputeInstance               `json:"compute_instances"`
	HealthcareDatasets   []*tfconfig.HealthcareDataset             `json:"healthcare_datasets"`
	IAMCustomRoles       []*tfconfig.ProjectIAMCustomRole          `json:"project_iam_custom_roles"`
	IAMMembers           *tfconfig.ProjectIAMMembers               `json:"project_iam_members"`
	NotificationChannels []*tfconfig.MonitoringNotificationChannel `json:"monitoring_notification_channels"`
	PubsubTopics         []*tfconfig.PubsubTopic                   `json:"pubsub_topics"`
	Services             *tfconfig.ProjectServices                 `json:"project_services"`
	ResourceManagerLiens []*tfconfig.ResourceManagerLien           `json:"resource_manager_liens"`
	ServiceAccounts      []*tfconfig.ServiceAccount                `json:"service_accounts"`
	StorageBuckets       []*tfconfig.StorageBucket                 `json:"storage_buckets"`

	BinauthzPolicy *BinAuthz `json:"binauthz"`

	AuditLogs *struct {
		LogsBQDataset BigqueryDataset `json:"logs_bq_dataset"`
		LogsGCSBucket *GCSBucket      `json:"logs_gcs_bucket"`
	} `json:"audit_logs"`

	Audit struct {
		LogsBigqueryDataset *tfconfig.BigqueryDataset `json:"logs_bigquery_dataset"`
		LogsStorageBucket   *tfconfig.StorageBucket   `json:"logs_storage_bucket"`
	} `json:"audit"`

	// The following vars are set through helpers and not directly through the user defined config.
	GeneratedFields *GeneratedFields `json:"-"`
	BQLogSink       *LogSink         `json:"-"`
	// TODO: replace DM log sink with TF once DM is deprecated.
	BQLogSinkTF *tfconfig.LoggingSink `json:"-"`
	Metrics     []*Metric             `json:"-"`

	IAMAuditConfig        *tfconfig.ProjectIAMAuditConfig   `json:"-"`
	DefaultAlertPolicies  []*tfconfig.MonitoringAlertPolicy `json:"-"`
	DefaultLoggingMetrics []*tfconfig.LoggingMetric         `json:"-"`
}

// Init initializes the config and all its projects.
func (c *Config) Init(genFields *AllGeneratedFields) error {
	if err := c.validate(); err != nil {
		return fmt.Errorf("failed to validate config: %v", err)
	}

	if genFields == nil {
		genFields = &AllGeneratedFields{}
	}
	if genFields.Projects == nil {
		genFields.Projects = make(map[string]*GeneratedFields)
	}
	if genFields.Forseti == nil {
		genFields.Forseti = &ForsetiServiceInfo{}
	}
	c.AllGeneratedFields = genFields
	if err := c.initForseti(); err != nil {
		return fmt.Errorf("failed to init forseti: %v", err)
	}

	ids := make(map[string]bool)
	for _, p := range c.AllProjects() {
		if ids[p.ID] {
			return fmt.Errorf("project %q defined more than once", p.ID)
		}
		ids[p.ID] = true
		if c.AllGeneratedFields.Projects[p.ID] == nil {
			c.AllGeneratedFields.Projects[p.ID] = &GeneratedFields{}
		}
		p.GeneratedFields = c.AllGeneratedFields.Projects[p.ID]
		if err := p.Init(c.ProjectForDevops(p), c.ProjectForAuditLogs(p)); err != nil {
			return fmt.Errorf("failed to init project %q: %v", p.ID, err)
		}
	}
	return nil
}

// validate validates the config.
func (c *Config) validate() error {
	// Enforce allowed_apis in overall project config.
	if len(c.Overall.AllowedAPIs) == 0 {
		return nil
	}
	allowedAPIs := make(map[string]bool)
	for _, a := range c.Overall.AllowedAPIs {
		allowedAPIs[a] = true
	}
	for _, p := range c.AllProjects() {
		for _, a := range p.EnabledAPIs {
			if !allowedAPIs[a] {
				return fmt.Errorf("project %q wants to enable API %q, which is not in the allowed APIs list", p.ID, a)
			}
		}
	}
	return nil
}

// AllFolders returns all folder ids in this config.
func (c *Config) AllFolders() []string {
	var ids []string
	if c.Overall.FolderID != "" {
		ids = append(ids, c.Overall.FolderID)
	}
	for _, p := range c.AllProjects() {
		if p.FolderID != "" {
			ids = append(ids, p.FolderID)
		}
	}
	return ids
}

// AllProjects returns all projects in this config.
// This includes Audit, Forseti and all data hosting projects.
func (c *Config) AllProjects() []*Project {
	ps := make([]*Project, 0, len(c.Projects))
	if c.Devops != nil {
		ps = append(ps, c.Devops.Project)
	}
	if c.AuditLogsProject != nil {
		ps = append(ps, c.AuditLogsProject)
	}
	if c.Forseti != nil {
		ps = append(ps, c.Forseti.Project)
	}
	ps = append(ps, c.Projects...)
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].ID < ps[j].ID
	})
	return ps
}

// ProjectForDevops is a helper function to get the devops project for the given project.
// Return the devops project if it exists, else return the project itself (to store devops resources locally).
func (c *Config) ProjectForDevops(p *Project) *Project {
	if c.Devops != nil {
		return c.Devops.Project
	}
	return p
}

// ProjectForAuditLogs is a helper function to get the audit logs project for the given project.
// Return the remote audit logs project if it exists, else return the project itself (to store audit logs locally).
func (c *Config) ProjectForAuditLogs(p *Project) *Project {
	if c.AuditLogsProject != nil {
		return c.AuditLogsProject
	}
	return p
}

func (c *Config) initForseti() error {
	if c.Forseti == nil {
		return nil
	}
	if c.Forseti.Properties == nil {
		c.Forseti.Properties = new(ForsetiProperties)
	}
	if err := c.Forseti.Properties.Init(); err != nil {
		return fmt.Errorf("failed to init forseti properties: %v", err)
	}
	p := c.Forseti.Properties
	p.ProjectID = c.Forseti.Project.ID
	p.Domain = c.Overall.Domain

	var resources []string
	if c.Overall.OrganizationID != "" {
		resources = append(resources, "organizations/"+c.Overall.OrganizationID)
	}
	for _, f := range c.AllFolders() {
		resources = append(resources, "folders/"+f)
	}
	p.CompositeRootResources = resources
	return nil
}

// Init initializes a project and all its resources.
// Audit Logs Project should either be a remote project or nil.
func (p *Project) Init(devopsProject, auditLogsProject *Project) error {
	if p.GeneratedFields == nil {
		p.GeneratedFields = &GeneratedFields{}
	}

	// init the state bucket outside the regular terraform resoures since forseti projects will
	// still define a state bucket even when not enabling terraform.
	if sb := p.TerraformConfig.StateBucket; sb != nil {
		if err := sb.Init(devopsProject.ID); err != nil {
			return fmt.Errorf("failed to init terraform state bucket: %v", err)
		}
	}

	if EnableTerraform {
		sb := p.TerraformConfig.StateBucket
		if sb == nil {
			return errors.New("state_storage_bucket must be set when terraform is enabled")
		}

		// State bucket will be deployed in the same deployment as the project.
		sb.DependsOn = append(sb.DependsOn, "google_project.project")
		return p.initTerraform(auditLogsProject)
	}

	if err := p.initAuditResources(auditLogsProject); err != nil {
		return fmt.Errorf("failed to init audit resources: %v", err)
	}

	for _, r := range p.DeploymentManagerResources() {
		if err := r.Init(); err != nil {
			return fmt.Errorf("failed to init: %v, %+v", err, r)
		}
	}

	if err := p.initDataResources(); err != nil {
		return fmt.Errorf("failed to init data resources: %v", err)
	}

	if err := p.addBaseResources(); err != nil {
		return fmt.Errorf("failed to add base resources: %v", err)
	}
	return nil
}

func (p *Project) initAuditResources(auditProject *Project) error {
	p.BQLogSink = &LogSink{
		LogSinkProperties: LogSinkProperties{
			Sink:                 "audit-logs-to-bigquery",
			Destination:          fmt.Sprintf("bigquery.googleapis.com/projects/%s/datasets/%s", auditProject.ID, p.AuditLogs.LogsBQDataset.Name()),
			Filter:               `logName:"logs/cloudaudit.googleapis.com"`,
			UniqueWriterIdentity: true,
		},
	}

	if err := p.AuditLogs.LogsBQDataset.Init(); err != nil {
		return fmt.Errorf("failed to init logs bq dataset: %v", err)
	}

	accesses := []*Access{
		{Role: "OWNER", GroupByEmail: auditProject.OwnersGroup},
		{Role: "READER", GroupByEmail: p.AuditorsGroup},
	}

	// Note: if there is no log sink SA it means the project hasn't been deployed.
	// The SA will be set once the project gets deployed (apply.Apply).
	if p.GeneratedFields.LogSinkServiceAccount != "" {
		accesses = append(accesses, &Access{Role: "WRITER", UserByEmail: p.GeneratedFields.LogSinkServiceAccount})
	}
	p.AuditLogs.LogsBQDataset.Accesses = accesses

	if p.AuditLogs.LogsGCSBucket == nil {
		return nil
	}

	if err := p.AuditLogs.LogsGCSBucket.Init(); err != nil {
		return fmt.Errorf("faild to init logs gcs bucket: %v", err)
	}

	p.AuditLogs.LogsGCSBucket.Bindings = []Binding{
		{Role: "roles/storage.admin", Members: []string{"group:" + auditProject.OwnersGroup}},
		{Role: "roles/storage.objectCreator", Members: []string{accessLogsWriter}},
		{Role: "roles/storage.objectViewer", Members: []string{"group:" + p.AuditorsGroup}},
	}

	return nil
}

func (p *Project) initDataResources() error {
	for _, d := range p.Resources.BQDatasets {
		// Note: duplicate accesses are de-duplicated by deployment manager.
		roleAndGroups := []struct {
			Role   string
			Groups []string
		}{
			{"OWNER", []string{p.OwnersGroup}},
			{"WRITER", p.DataReadWriteGroups},
			{"READER", p.DataReadOnlyGroups},
		}

		for _, rg := range roleAndGroups {
			for _, g := range rg.Groups {
				d.Accesses = append(d.Accesses, &Access{
					Role:         rg.Role,
					GroupByEmail: g,
				})
			}
		}
	}

	appendGroupPrefix := func(ss ...string) []string {
		res := make([]string, 0, len(ss))
		for _, s := range ss {
			res = append(res, "group:"+s)
		}
		return res
	}

	for _, b := range p.Resources.GCSBuckets {
		// Note: duplicate bindings are de-duplicated by deployment manager.
		bindings := []Binding{
			{Role: "roles/storage.admin", Members: appendGroupPrefix(p.OwnersGroup)},
		}
		if len(p.DataReadWriteGroups) > 0 {
			bindings = append(bindings, Binding{
				Role: "roles/storage.objectAdmin", Members: appendGroupPrefix(p.DataReadWriteGroups...),
			})
		}
		if len(p.DataReadOnlyGroups) > 0 {
			bindings = append(bindings, Binding{
				Role: "roles/storage.objectViewer", Members: appendGroupPrefix(p.DataReadOnlyGroups...),
			})
		}
		b.Bindings = MergeBindings(append(bindings, b.Bindings...)...)

		// TODO: this should always be true (data buckets should imply log bucket exists).
		if p.AuditLogs.LogsGCSBucket != nil {
			if b.Logging == nil {
				b.Logging = new(logging)
			}
			b.Logging.LogBucket = p.AuditLogs.LogsGCSBucket.Name()
		}
	}

	for _, ps := range p.Resources.Pubsubs {
		defaultBindings := []Binding{
			{"roles/pubsub.editor", appendGroupPrefix(p.DataReadWriteGroups...)},
			{"roles/pubsub.viewer", appendGroupPrefix(p.DataReadOnlyGroups...)},
		}

		for _, s := range ps.Subscriptions {
			s.Bindings = MergeBindings(append(defaultBindings, s.Bindings...)...)
		}
	}

	return nil
}

// addBaseResources adds resources not set by the raw yaml config in the project (i.e. not configured by the user).
func (p *Project) addBaseResources() error {
	p.Resources.IAMPolicies = append(p.Resources.IAMPolicies, &IAMPolicy{
		IAMPolicyName: "required-project-bindings",
		IAMPolicyProperties: IAMPolicyProperties{Bindings: []Binding{
			{Role: "roles/owner", Members: []string{"group:" + p.OwnersGroup}},
			{Role: "roles/iam.securityReviewer", Members: []string{"group:" + p.AuditorsGroup}},
		}},
	})
	defaultMetrics := []*Metric{
		&Metric{
			MetricProperties: MetricProperties{
				MetricName:      BQSettingChangeMetricName,
				Description:     "Count of bigquery permission changes.",
				Filter:          `resource.type="bigquery_resource" AND protoPayload.methodName="datasetservice.update"`,
				Descriptor:      unexpectedUserDescriptor,
				LabelExtractors: principalEmailLabelExtractor,
			},
		},
		&Metric{
			MetricProperties: MetricProperties{
				MetricName:      IAMChangeMetricName,
				Description:     "Count of IAM policy changes.",
				Filter:          `protoPayload.methodName="SetIamPolicy" OR protoPayload.methodName:".setIamPolicy"`,
				Descriptor:      unexpectedUserDescriptor,
				LabelExtractors: principalEmailLabelExtractor,
			},
		},
		&Metric{
			MetricProperties: MetricProperties{
				MetricName:  BucketPermissionChangeMetricName,
				Description: "Count of GCS permissions changes.",
				Filter: `resource.type=gcs_bucket AND protoPayload.serviceName=storage.googleapis.com AND
(protoPayload.methodName=storage.setIamPermissions OR protoPayload.methodName=storage.objects.update)`,
				Descriptor:      unexpectedUserDescriptor,
				LabelExtractors: principalEmailLabelExtractor,
			},
		},
	}
	excludeMetricPrincipleEmails, err := template.New("excludeEmails").Parse(` AND
protoPayload.authenticationInfo.principalEmail!=({{.ExpectedAccounts}})`)
	if err != nil {
		return err
	}
	for index, dm := range defaultMetrics {
		if violationExceptions, ok := p.ViolationExceptions[dm.MetricProperties.MetricName]; ok {
			var buf bytes.Buffer
			data := struct {
				ExpectedAccounts string
			}{
				strings.Join(violationExceptions, " AND "),
			}
			if err := excludeMetricPrincipleEmails.Execute(&buf, data); err != nil {
				return fmt.Errorf("failed to execute filter template: %v", err)
			}
			defaultMetrics[index].MetricProperties.Filter = dm.MetricProperties.Filter + buf.String()
		}
	}
	p.Metrics = append(p.Metrics, defaultMetrics...)

	metricFilterTemplate, err := template.New("metricFilter").Parse(`resource.type=gcs_bucket AND
logName=projects/{{.Project.ID}}/logs/cloudaudit.googleapis.com%2Fdata_access AND
protoPayload.resourceName=projects/_/buckets/{{.Bucket.Name}} AND
protoPayload.status.code!=7 AND
protoPayload.authenticationInfo.principalEmail!=({{.ExpectedUsers}})`)
	if err != nil {
		return err
	}

	for _, b := range p.Resources.GCSBuckets {
		if len(b.ExpectedUsers) == 0 {
			continue
		}

		var buf bytes.Buffer
		data := struct {
			Project       *Project
			Bucket        *GCSBucket
			ExpectedUsers string
		}{
			p,
			b,
			strings.Join(b.ExpectedUsers, " AND "),
		}
		if err := metricFilterTemplate.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute filter template: %v", err)
		}

		p.Metrics = append(p.Metrics, &Metric{
			MetricProperties: MetricProperties{
				MetricName:      BucketUnexpectedAccessMetricPrefix + b.Name(),
				Description:     "Count of unexpected data access to " + b.Name(),
				Filter:          buf.String(),
				Descriptor:      unexpectedUserDescriptor,
				LabelExtractors: principalEmailLabelExtractor,
			},
			dependencies: []string{b.Name()},
		})
	}
	return nil
}

// Resource is an interface that must be implemented by all concrete resource implementations.
type Resource interface {
	Init() error
	Name() string
}

// DeploymentManagerResources gets all deployment manager data resources in this project.
func (p *Project) DeploymentManagerResources() []Resource {
	rs := []Resource{p.BQLogSink}

	for _, r := range p.Metrics {
		rs = append(rs, r)
	}

	prs := p.Resources

	for _, r := range prs.BQDatasets {
		rs = append(rs, r)
	}
	for _, r := range prs.CHCDatasets {
		rs = append(rs, r)
	}
	for _, r := range prs.CloudRouter {
		r.TmplPath = "config/templates/cloud_router/cloud_router.py"
		rs = append(rs, r)
	}
	for _, r := range prs.GCEFirewalls {
		r.TmplPath = "config/templates/firewall/firewall.py"
		rs = append(rs, r)
	}
	for _, r := range prs.GCEInstances {
		rs = append(rs, r)
	}
	for _, r := range prs.GCSBuckets {
		rs = append(rs, r)
	}
	for _, r := range prs.GKEClusters {
		rs = append(rs, r)
	}
	for _, r := range prs.IAMCustomRoles {
		rs = append(rs, r)
	}
	for _, r := range prs.IAMPolicies {
		rs = append(rs, r)
	}
	for _, r := range prs.IPAddresses {
		r.TmplPath = "config/templates/ip_reservation/ip_address.py"
		rs = append(rs, r)
	}
	for _, r := range prs.Routes {
		r.TmplPath = "config/templates/route/single_route.py"
		rs = append(rs, r)
	}
	for _, r := range prs.ServiceAccounts {
		rs = append(rs, r)
	}
	for _, r := range prs.Pubsubs {
		rs = append(rs, r)
	}
	for _, r := range prs.VPCNetworks {
		r.TmplPath = "config/templates/network/network.py"
		rs = append(rs, r)
	}
	for _, r := range prs.VPNs {
		r.TmplPath = "config/templates/vpn/vpn.py"
		rs = append(rs, r)
	}
	return rs
}
