// Package cft provides utilities to deploy CFT resources.
package cft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
)

// accessLogsWriter is the access logs writer.
// https://cloud.google.com/storage/docs/access-logs#delivery.
const accessLogsWriter = "group:cloud-storage-analytics@google.com"

// Config represents a (partial)f representation of a projects YAML file.
// Only the required fields are present. See project_config.yaml.schema for details.
// TODO: support new generated_fields
type Config struct {
	Overall struct {
		Domain         string   `json:"domain"`
		OrganizationID string   `json:"organization_id"`
		FolderID       string   `json:"folder_id"`
		AllowedAPIs    []string `json:"allowed_apis"`
	} `json:"overall"`
	AuditLogsProject *Project `json:"audit_logs_project"`
	Forseti          *struct {
		Project *Project `json:"project"`
	} `json:"forseti"`
	Projects           []*Project         `json:"projects"`
	AllGeneratedFields AllGeneratedFields `json:"generated_fields"`
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
	EnabledAPIs         []string `json:"enabled_apis"`

	Resources struct {
		// Deployment manager resources
		BQDatasets     []*BigqueryDatasetPair `json:"bq_datasets"`
		GCEFirewalls   []*GCEFirewallPair     `json:"gce_firewalls"`
		GCEInstances   []*GCEInstancePair     `json:"gce_instances"`
		GCSBuckets     []*GCSBucketPair       `json:"gcs_buckets"`
		GKEClusters    []*GKEClusterPair      `json:"gke_clusters"`
		IAMCustomRoles []*IAMCustomRolePair   `json:"iam_custom_roles"`
		IAMPolicies    []*IAMPolicyPair       `json:"iam_policies"`
		Pubsubs        []*PubsubPair          `json:"pubsubs"`

		// Kubectl resources
		GKEWorkloads []*GKEWorkload `json:"gke_workloads"`
	} `json:"resources"`

	AuditLogs *struct {
		// While audit resources use CFT templates under the hood, we only allow them to set
		// a controlled subset of fields as these resources are managed more tightly by the DPT scripts.
		// Thus, don't implement them as a resource pair as we don't need to save the raw form.
		LogsBQDataset BigqueryDataset `json:"logs_bq_dataset"`
		LogsGCSBucket *GCSBucket      `json:"logs_gcs_bucket"`
	} `json:"audit_logs"`

	// The following vars are set through helpers and not directly through the user defined config.
	GeneratedFields  *GeneratedFields   `json:"-"`
	BQLogSink        *LogSink           `json:"-"`
	DefaultResources []*DefaultResource `json:"-"`
	Metrics          []*Metric          `json:"-"`
}

// BigqueryDatasetPair pairs a raw dataset with its parsed version.
type BigqueryDatasetPair struct {
	json.RawMessage
	Parsed BigqueryDataset
}

// GCEFirewallPair pairs a raw firewall with its parsed version.
type GCEFirewallPair struct {
	json.RawMessage
	Parsed DefaultResource
}

// GCEInstancePair pairs a raw instance with its parsed version.
type GCEInstancePair struct {
	json.RawMessage
	Parsed GCEInstance
}

// GCSBucketPair pairs a raw bucket with its parsed version.
type GCSBucketPair struct {
	json.RawMessage
	Parsed GCSBucket
}

// GKEClusterPair pairs a raw cluster with its parsed version.
type GKEClusterPair struct {
	json.RawMessage
	Parsed GKECluster `json:"-"`
}

// IAMCustomRolePair pairs a raw custom role with its parsed version.
type IAMCustomRolePair struct {
	json.RawMessage
	Parsed IAMCustomRole
}

// IAMPolicyPair pairs a raw iam policy with its parsed version.
type IAMPolicyPair struct {
	json.RawMessage
	Parsed IAMPolicy
}

// PubsubPair pairs a raw pubsub with its parsed version.
type PubsubPair struct {
	json.RawMessage
	Parsed Pubsub
}

// Init initializes the config and all its projects.
func (c *Config) Init() error {
	for _, p := range c.AllProjects() {
		p.GeneratedFields = c.AllGeneratedFields.Projects[p.ID]
		if err := p.Init(c.AuditLogsProject); err != nil {
			return fmt.Errorf("failed to init project %q: %v", p.ID, err)
		}
	}
	return nil
}

// AllProjects returns all projects in this config.
// This includes Audit, Forseti and all data hosting projects.
func (c *Config) AllProjects() []*Project {
	ps := make([]*Project, 0, len(c.Projects))
	if c.AuditLogsProject != nil {
		ps = append(ps, c.AuditLogsProject)
	}
	if c.Forseti != nil {
		ps = append(ps, c.Forseti.Project)
	}
	ps = append(ps, c.Projects...)
	return ps
}

// ProjectForAuditLogs is a helper function to get the audit logs project for the given project.
// Return the remote audit logs project if it exists, else return the project itself (to store audit logs locally).
func (c *Config) ProjectForAuditLogs(p *Project) *Project {
	if c.AuditLogsProject != nil {
		return c.AuditLogsProject
	}
	return p
}

// Init initializes a project and all its resources.
// Audit Logs Project should either be a remote project or nil.
func (p *Project) Init(auditLogsProject *Project) error {
	if p.GeneratedFields == nil {
		p.GeneratedFields = &GeneratedFields{}
	}

	if err := p.initAuditResources(auditLogsProject); err != nil {
		return fmt.Errorf("failed to init audit resources: %v", err)
	}

	for _, pair := range p.ResourcePairs() {
		if len(pair.Raw) > 0 {
			if err := json.Unmarshal(pair.Raw, pair.Parsed); err != nil {
				return err
			}
		}
		if err := pair.Parsed.Init(p); err != nil {
			return fmt.Errorf("failed to init: %v, %+v", err, pair.Parsed)
		}
	}

	if err := p.addBaseResources(); err != nil {
		return fmt.Errorf("failed to add base resources: %v", err)
	}
	return nil
}

func (p *Project) initAuditResources(auditProject *Project) error {
	if auditProject == nil {
		auditProject = p
	}

	p.BQLogSink = &LogSink{
		LogSinkProperties: LogSinkProperties{
			Sink:                 "audit-logs-to-bigquery",
			Destination:          fmt.Sprintf("bigquery.googleapis.com/projects/%s/datasets/%s", auditProject.ID, p.AuditLogs.LogsBQDataset.Name()),
			Filter:               `logName:"logs/cloudaudit.googleapis.com"`,
			UniqueWriterIdentity: true,
		},
	}

	if err := p.AuditLogs.LogsBQDataset.Init(p); err != nil {
		return fmt.Errorf("failed to init logs bq dataset: %v", err)
	}

	accesses := []Access{
		{Role: "OWNER", GroupByEmail: auditProject.OwnersGroup},
		{Role: "READER", GroupByEmail: p.AuditorsGroup},
	}

	if p.GeneratedFields.LogSinkServiceAccount != "" {
		accesses = append(accesses, Access{Role: "WRITER", UserByEmail: p.GeneratedFields.LogSinkServiceAccount})
	}
	p.AuditLogs.LogsBQDataset.Accesses = accesses

	if p.AuditLogs.LogsGCSBucket == nil {
		return nil
	}

	if err := p.AuditLogs.LogsGCSBucket.Init(p); err != nil {
		return fmt.Errorf("faild to init logs gcs bucket: %v", err)
	}

	p.AuditLogs.LogsGCSBucket.Bindings = []Binding{
		{Role: "roles/storage.admin", Members: []string{"group:" + auditProject.OwnersGroup}},
		{Role: "roles/storage.objectCreator", Members: []string{accessLogsWriter}},
		{Role: "roles/storage.objectViewer", Members: []string{"group:" + p.AuditorsGroup}},
	}

	return nil
}

// addBaseResources adds resources not set by the raw yaml config in the project (i.e. not configured by the user).
func (p *Project) addBaseResources() error {
	p.DefaultResources = append(p.DefaultResources, &DefaultResource{
		DefaultResourceProperties: DefaultResourceProperties{ResourceName: "enable-all-audit-log-policies"},
		templatePath:              "deploy/templates/audit_log_config.py",
	})

	p.Resources.IAMPolicies = append(p.Resources.IAMPolicies, &IAMPolicyPair{
		Parsed: IAMPolicy{
			IAMPolicyName: "required-project-bindings",
			IAMPolicyProperties: IAMPolicyProperties{Bindings: []Binding{
				{Role: "roles/owner", Members: []string{"group:" + p.OwnersGroup}},
				{Role: "roles/iam.securityReviewer", Members: []string{"group:" + p.AuditorsGroup}},
			}}},
	})
	p.Metrics = append(p.Metrics,
		&Metric{
			MetricProperties: MetricProperties{
				MetricName:      "bigquery-settings-change-count",
				Description:     "Count of bigquery permission changes.",
				Filter:          `resource.type="bigquery_resource" AND protoPayload.methodName="datasetservice.update"`,
				Descriptor:      unexpectedUserDescriptor,
				LabelExtractors: principalEmailLabelExtractor,
			},
		},
		&Metric{
			MetricProperties: MetricProperties{
				MetricName:      "iam-policy-change-count",
				Description:     "Count of IAM policy changes.",
				Filter:          `protoPayload.methodName="SetIamPolicy" OR protoPayload.methodName:".setIamPolicy"`,
				Descriptor:      unexpectedUserDescriptor,
				LabelExtractors: principalEmailLabelExtractor,
			},
		},
		&Metric{
			MetricProperties: MetricProperties{
				MetricName:  "bucket-permission-change-count",
				Description: "Count of GCS permissions changes.",
				Filter: `resource.type=gcs_bucket AND protoPayload.serviceName=storage.googleapis.com AND
(protoPayload.methodName=storage.setIamPermissions OR protoPayload.methodName=storage.objects.update)`,
				Descriptor:      unexpectedUserDescriptor,
				LabelExtractors: principalEmailLabelExtractor,
			},
		})

	metricFilterTemplate, err := template.New("metricFilter").Parse(`resource.type=gcs_bucket AND
logName=projects/{{.Project.ID}}/logs/cloudaudit.googleapis.com%2Fdata_access AND
protoPayload.resourceName=projects/_/buckets/{{.Bucket.Name}} AND
protoPayload.status.code!=7 AND
protoPayload.authenticationInfo.principalEmail!=({{.ExpectedUsers}})`)
	if err != nil {
		return err
	}

	for _, pair := range p.Resources.GCSBuckets {
		b := pair.Parsed
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
			&b,
			strings.Join(b.ExpectedUsers, " AND "),
		}
		if err := metricFilterTemplate.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute filter template: %v", err)
		}

		p.Metrics = append(p.Metrics, &Metric{
			MetricProperties: MetricProperties{
				MetricName:      "unexpected-access-" + b.Name(),
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

// parsedResource is an interface that must be implemented by all concrete resource implementations.
type parsedResource interface {
	Init(*Project) error
	Name() string
}

// ResourcePairs gets all deployment manager data resources in this project.
// The raw field will be set for resources that retain their original raw user input.
// This raw form should be merged with the parsed form before being deployed.
func (p *Project) ResourcePairs() []ResourcePair {
	pairs := []ResourcePair{{Parsed: p.BQLogSink}}

	for _, r := range p.DefaultResources {
		pairs = append(pairs, ResourcePair{Parsed: r})
	}
	for _, r := range p.Metrics {
		pairs = append(pairs, ResourcePair{Parsed: r})
	}

	appendPair := func(raw json.RawMessage, parsed parsedResource) {
		pairs = append(pairs, ResourcePair{raw, parsed})
	}
	appendDefaultResPair := func(raw json.RawMessage, parsed *DefaultResource, path string) {
		parsed.templatePath = path
		appendPair(raw, parsed)
	}

	rs := p.Resources

	for _, r := range rs.BQDatasets {
		appendPair(r.RawMessage, &r.Parsed)
	}
	for _, r := range rs.GCEFirewalls {
		appendDefaultResPair(r.RawMessage, &r.Parsed, "deploy/cft/templates/firewall/firewall.py")
	}
	for _, r := range rs.GCEInstances {
		appendPair(r.RawMessage, &r.Parsed)
	}
	for _, r := range rs.GCSBuckets {
		appendPair(r.RawMessage, &r.Parsed)
	}
	for _, r := range rs.GKEClusters {
		appendPair(r.RawMessage, &r.Parsed)
	}
	for _, r := range rs.IAMCustomRoles {
		appendPair(r.RawMessage, &r.Parsed)
	}
	for _, r := range rs.IAMPolicies {
		appendPair(r.RawMessage, &r.Parsed)
	}
	for _, r := range rs.Pubsubs {
		appendPair(r.RawMessage, &r.Parsed)
	}
	return pairs
}
