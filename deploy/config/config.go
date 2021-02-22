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
	"errors"
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
)

// EnableTerraform determines whether terraform will be enabled or not.
// Note: The terraform state bucket does not respect this var as it is required currently for Forseti projects.
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
	ID                  string            `json:"project_id"`
	BillingAccount      string            `json:"billing_account"`
	FolderID            string            `json:"folder_id"`
	OwnersGroup         string            `json:"owners_group"`
	AuditorsGroup       string            `json:"auditors_group"`
	DataReadWriteGroups []string          `json:"data_readwrite_groups"`
	DataReadOnlyGroups  []string          `json:"data_readonly_groups"`
	Labels              map[string]string `json:"labels"`

	DevopsConfig struct {
		StateBucket *tfconfig.StorageBucket `json:"state_storage_bucket"`
	} `json:"devops"`

	CreateDeletionLien    bool                `json:"create_deletion_lien"`
	EnabledAPIs           []string            `json:"enabled_apis"`
	ViolationExceptions   map[string][]string `json:"violation_exceptions"`
	StackdriverAlertEmail string              `json:"stackdriver_alert_email"`

	// Terraform resources
	BigqueryDatasets     []*tfconfig.BigqueryDataset               `json:"bigquery_datasets"`
	CloudBuildTriggers   []*tfconfig.CloudBuildTrigger             `json:"cloudbuild_triggers"`
	ComputeFirewalls     []*tfconfig.ComputeFirewall               `json:"compute_firewalls"`
	ComputeImages        []*tfconfig.ComputeImage                  `json:"compute_images"`
	ComputeInstances     []*tfconfig.ComputeInstance               `json:"compute_instances"`
	DataFusionInstances  []*tfconfig.DataFusionInstance            `json:"data_fusion_instances"`
	HealthcareDatasets   []*tfconfig.HealthcareDataset             `json:"healthcare_datasets"`
	IAMCustomRoles       []*tfconfig.ProjectIAMCustomRole          `json:"project_iam_custom_roles"`
	IAMMembers           *tfconfig.ProjectIAMMembers               `json:"project_iam_members"`
	NotificationChannels []*tfconfig.MonitoringNotificationChannel `json:"monitoring_notification_channels"`
	PubsubTopics         []*tfconfig.PubsubTopic                   `json:"pubsub_topics"`
	Services             *tfconfig.ProjectServices                 `json:"project_services"`
	ResourceManagerLiens []*tfconfig.ResourceManagerLien           `json:"resource_manager_liens"`
	ServiceAccounts      []*tfconfig.ServiceAccount                `json:"service_accounts"`
	SpannerInstances     []*tfconfig.SpannerInstance               `json:"spanner_instances"`
	StorageBuckets       []*tfconfig.StorageBucket                 `json:"storage_buckets"`

	Audit struct {
		LogsBigqueryDataset *tfconfig.BigqueryDataset `json:"logs_bigquery_dataset"`
		LogsStorageBucket   *tfconfig.StorageBucket   `json:"logs_storage_bucket"`
	} `json:"audit"`

	TerraformDeployments struct {
		Resources struct {
			Config map[string]interface{} `json:"config"`
		} `json:"resources"`
	} `json:"terraform_deployments"`

	// The following vars are set through helpers and not directly through the user defined config.
	GeneratedFields *GeneratedFields      `json:"-"`
	BQLogSinkTF     *tfconfig.LoggingSink `json:"-"`

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

	p.CompositeRootResources = c.Forseti.Properties.CompositeRootResources
	if len(p.CompositeRootResources) == 0 {
		if c.Overall.OrganizationID != "" {
			p.CompositeRootResources = append(p.CompositeRootResources, "organizations/"+c.Overall.OrganizationID)
		}
		for _, f := range c.AllFolders() {
			p.CompositeRootResources = append(p.CompositeRootResources, "folders/"+f)
		}
	}
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
	if sb := p.DevopsConfig.StateBucket; sb != nil {
		if err := sb.Init(devopsProject.ID); err != nil {
			return fmt.Errorf("failed to init terraform state bucket: %v", err)
		}
		// TODO: Uncomment this once we can support importing iam members, else this resource will always be created.
		// sb.IAMMembers = append(sb.IAMMembers, &tfconfig.StorageIAMMember{
		// 	Role:   "roles/storage.admin",
		// 	Member: "group:" + p.OwnersGroup,
		// })
	}

	sb := p.DevopsConfig.StateBucket
	if sb == nil {
		return errors.New("state_storage_bucket must be set when terraform is enabled")
	}
	return p.initTerraform(auditLogsProject)
}
