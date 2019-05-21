// Package cft provides utilities to deploy CFT resources.
package cft

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/healthcare/deploy/deploymentmanager"
)

// accessLogsWriter is the access logs writer.
// https://cloud.google.com/storage/docs/access-logs#delivery.
const accessLogsWriter = "group:cloud-storage-analytics@google.com"

const (
	deploymentNamePrefix   = "data-protect-toolkit"
	auditDeploymentName    = deploymentNamePrefix + "-audit"
	resourceDeploymentName = deploymentNamePrefix + "-resources"
)

// The following vars are stubbed in tests.
var (
	cmdRun            = (*exec.Cmd).Run
	cmdCombinedOutput = (*exec.Cmd).CombinedOutput
	upsertDeployment  = deploymentmanager.Upsert
)

// Config represents a (partial) representation of a projects YAML file.
// Only the required fields are present. See project_config.yaml.schema for details.
// TODO: move config to its own package
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
	EditorsGroup        string   `json:"editors_group"`
	DataReadWriteGroups []string `json:"data_readwrite_groups"`
	DataReadOnlyGroups  []string `json:"data_readonly_groups"`
	EnabledAPIs         []string `json:"enabled_apis"`

	Resources struct {
		// Deployment manager resources
		BQDatasets     []*BigqueryDatasetPair `json:"bq_datasets"`
		Firewalls      []*FirewallPair        `json:"firewalls"`
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

	// Passed through by config so should not be parsed by json.
	GeneratedFields *GeneratedFields `json:"-"`

	// Set in init so should not be parsed by json.
	BQLogSink *LogSink `json:"-"`
}

// BigqueryDatasetPair pairs a raw dataset with its parsed version.
type BigqueryDatasetPair struct {
	json.RawMessage
	Parsed BigqueryDataset
}

// FirewallPair pairs a raw firewall with its parsed version.
type FirewallPair struct {
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

	for _, pair := range p.resourcePairs() {
		if len(pair.raw) > 0 {
			if err := json.Unmarshal(pair.raw, pair.parsed); err != nil {
				return err
			}
		}
		if err := pair.parsed.Init(p); err != nil {
			return fmt.Errorf("failed to init: %v, %+v", err, pair.parsed)
		}
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

func (p *Project) resourcePairs() []resourcePair {
	pairs := []resourcePair{{parsed: p.BQLogSink}}

	for _, res := range defaultResources {
		pairs = append(pairs, resourcePair{parsed: res})
	}
	appendPair := func(raw json.RawMessage, parsed parsedResource) {
		pairs = append(pairs, resourcePair{raw, parsed})
	}
	appendDefaultResPair := func(raw json.RawMessage, parsed *DefaultResource, path string) {
		parsed.templatePath = path
		appendPair(raw, parsed)
	}

	rs := p.Resources

	for _, r := range rs.BQDatasets {
		appendPair(r.RawMessage, &r.Parsed)
	}
	for _, r := range rs.Firewalls {
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

var defaultResources = []parsedResource{
	&DefaultResource{
		DefaultResourceProperties: DefaultResourceProperties{ResourceName: "enable-all-audit-log-policies"},
		templatePath:              "deploy/templates/audit_log_config.py",
	},
	&Metric{
		MetricProperties{
			MetricName:      "bigquery-settings-change-count",
			Description:     "Count of bigquery permission changes.",
			Filter:          `resource.type="bigquery_resource" AND protoPayload.methodName="datasetservice.update"`,
			Descriptor:      unexpectedUserDescriptor,
			LabelExtractors: principalEmailLabelExtractor,
		},
	},
	&Metric{
		MetricProperties{
			MetricName:      "iam-policy-change-count",
			Description:     "Count of IAM policy changes.",
			Filter:          `protoPayload.methodName="SetIamPolicy" OR protoPayload.methodName:".setIamPolicy"`,
			Descriptor:      unexpectedUserDescriptor,
			LabelExtractors: principalEmailLabelExtractor,
		},
	},
	&Metric{
		MetricProperties{
			MetricName:  "bucket-permission-change-count",
			Description: "Count of GCS permissions changes.",
			Filter: `resource.type=gcs_bucket AND protoPayload.serviceName=storage.googleapis.com AND
(protoPayload.methodName=storage.setIamPermissions OR protoPayload.methodName=storage.objects.update)`,
			Descriptor:      unexpectedUserDescriptor,
			LabelExtractors: principalEmailLabelExtractor,
		},
	},
}

// parsedResource is an interface that must be implemented by all concrete resource implementations.
type parsedResource interface {
	Init(*Project) error
	Name() string
}

// deploymentManagerTyper should be implemented by resources that are natively supported by the deployment manager service.
// Use this if there is no suitable CFT template for a resource and a custom template is not needed.
// See https://cloud.google.com/deployment-manager/docs/configuration/supported-resource-types for valid types.
type deploymentManagerTyper interface {
	DeploymentManagerType() string
}

// deploymentManagerPather should be implemented by resources that use a DM template to deploy.
// Use this if the resource wraps a CFT or custom template.
type deploymentManagerPather interface {
	TemplatePath() string
}

// depender is the interface that defines a method to get dependent resources.
type depender interface {
	DependentResources(*Project) ([]parsedResource, error)
}

// Deploy deploys the CFT resources in the project.
func Deploy(project, auditProject *Project) error {
	if err := deployResources(project); err != nil {
		return fmt.Errorf("failed to deploy resources: %v", err)
	}

	// If this is the initial deployment then there won't be a log sink service account
	// in the generated fields as it is deployed through the data-protect-toolkit-resources deployment.
	if project.GeneratedFields.LogSinkServiceAccount == "" {
		sinkSA, err := getLogSinkServiceAccount(project)
		if err != nil {
			return fmt.Errorf("failed to get log sink service account: %v", err)
		}
		project.GeneratedFields.LogSinkServiceAccount = sinkSA
		project.AuditLogs.LogsBQDataset.Accesses = append(project.AuditLogs.LogsBQDataset.Accesses, Access{
			Role: "WRITER", UserByEmail: sinkSA,
		})
	}
	if err := deployAudit(project, auditProject); err != nil {
		return fmt.Errorf("failed to deploy audit resources: %v", err)
	}

	if err := deployGKEWorkloads(project); err != nil {
		return fmt.Errorf("failed to deploy GKE workloads: %v", err)
	}
	return nil
}

func getLogSinkServiceAccount(project *Project) (string, error) {
	cmd := exec.Command("gcloud", "logging", "sinks", "describe", project.BQLogSink.Name(), "--format", "json", "--project", project.ID)

	out, err := cmdCombinedOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to query log sink service account from gcloud: %v\n%v", err, string(out))
	}

	type sink struct {
		WriterIdentity string `json:"writerIdentity"`
	}

	s := new(sink)
	if err := json.Unmarshal(out, s); err != nil {
		return "", fmt.Errorf("failed to unmarshal sink output: %v", err)
	}
	return strings.TrimPrefix(s.WriterIdentity, "serviceAccount:"), nil
}

func deployAudit(project, auditProject *Project) error {
	pairs := []resourcePair{
		{parsed: &project.AuditLogs.LogsBQDataset},
		{parsed: project.AuditLogs.LogsGCSBucket},
	}
	deployment, err := getDeployment(project, pairs)
	if err != nil {
		return err
	}

	// Append project ID to deployment name so each project has unique deployment if there is
	// a remote audit logs project.
	name := fmt.Sprintf("%s-%s", auditDeploymentName, project.ID)
	if err := upsertDeployment(name, deployment, auditProject.ID); err != nil {
		return fmt.Errorf("failed to deploy audit resources: %v", err)
	}
	return nil
}

func deployResources(project *Project) error {
	pairs := project.resourcePairs()
	if len(pairs) == 0 {
		log.Println("No resources to deploy.")
		return nil
	}
	deployment, err := getDeployment(project, pairs)
	if err != nil {
		return err
	}
	if err := upsertDeployment(resourceDeploymentName, deployment, project.ID); err != nil {
		return fmt.Errorf("failed to deploy deployment manager resources: %v", err)
	}
	return nil
}

func getDeployment(project *Project, pairs []resourcePair) (*deploymentmanager.Deployment, error) {
	deployment := &deploymentmanager.Deployment{}

	allImports := make(map[string]bool)

	for _, pair := range pairs {
		resources, importSet, err := getDeploymentResourcesAndImports(pair, project)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource and imports for %q: %v", pair.parsed.Name(), err)
		}
		deployment.Resources = append(deployment.Resources, resources...)

		var imports []string
		for imp := range importSet {
			imports = append(imports, imp)
		}
		sort.Strings(imports) // for stable ordering

		for _, imp := range imports {
			if !allImports[imp] {
				deployment.Imports = append(deployment.Imports, &deploymentmanager.Import{Path: imp})
			}
			allImports[imp] = true
		}
	}

	return deployment, nil
}

// getDeploymentResourcesAndImports gets the deploment resources and imports for the given resource pair.
// It also recursively adds the resoures and imports of any dependent resources.
func getDeploymentResourcesAndImports(pair resourcePair, project *Project) (resources []*deploymentmanager.Resource, importSet map[string]bool, err error) {
	importSet = make(map[string]bool)

	var typ string
	if typer, ok := pair.parsed.(deploymentManagerTyper); ok {
		typ = typer.DeploymentManagerType()
	} else if pather, ok := pair.parsed.(deploymentManagerPather); ok {
		var err error
		typ, err = filepath.Abs(pather.TemplatePath())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get absolute path for %q: %v", pather.TemplatePath(), err)
		}
		importSet[typ] = true
	} else {
		return nil, nil, fmt.Errorf("failed to get type of %+v", pair.parsed)
	}

	merged, err := pair.MergedPropertiesMap()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to merge raw map with parsed: %v", err)
	}

	resources = []*deploymentmanager.Resource{{
		Name:       pair.parsed.Name(),
		Type:       typ,
		Properties: merged,
	}}

	dr, ok := pair.parsed.(depender)
	if !ok { // doesn't implement dependent resources method so has no dependent resources
		return resources, importSet, nil
	}

	dependencies, err := dr.DependentResources(project)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get dependent resources for %q: %v", pair.parsed.Name(), err)
	}

	for _, dep := range dependencies {
		depPair := resourcePair{parsed: dep}
		depResources, depImports, err := getDeploymentResourcesAndImports(depPair, project)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get resources and imports for %q: %v", dep.Name(), err)
		}

		for _, res := range depResources {
			if res.Metadata == nil {
				res.Metadata = &deploymentmanager.Metadata{}
			}
			res.Metadata.DependsOn = append(res.Metadata.DependsOn, pair.parsed.Name())
		}
		resources = append(resources, depResources...)

		for imp := range depImports {
			importSet[imp] = true
		}
	}
	return resources, importSet, nil
}
