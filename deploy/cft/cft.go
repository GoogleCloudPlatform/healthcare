// Package cft provides utilities to deploy CFT resources.
package cft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

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

// deploymentManagerRoles are the roles granted to the DM service account.
var deploymentManagerRoles = []string{"roles/owner", "roles/storage.admin"}

// deploymentRetryWaitTime is the time to wait between retrying a deployment to allow for concurrent operations to finish.
const deploymentRetryWaitTime = time.Minute

// The following vars are stubbed in tests.
var (
	cmdOutput = func(cmd *exec.Cmd) ([]byte, error) {
		log.Printf("Running: %v", cmd.Args)
		cmd.Stderr = os.Stderr
		return cmd.Output()
	}
	cmdRun = func(cmd *exec.Cmd) error {
		log.Printf("Running: %v", cmd.Args)
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	upsertDeployment = deploymentmanager.Upsert
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

func (p *Project) resourcePairs() []resourcePair {
	pairs := []resourcePair{{parsed: p.BQLogSink}}

	for _, r := range p.DefaultResources {
		pairs = append(pairs, resourcePair{parsed: r})
	}
	for _, r := range p.Metrics {
		pairs = append(pairs, resourcePair{parsed: r})
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
	// Dependencies returns the name of the resource IDs to depend on.
	Dependencies() []string
}

// Deploy deploys the CFT resources in the project.
func Deploy(config *Config, project *Project) error {
	if err := grantDeploymentManagerAccess(project); err != nil {
		return fmt.Errorf("failed to grant deployment manager access to the project: %v", err)
	}

	// TODO: stop retrying once
	// https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/issues/17
	// is fixed.
	for i := 0; i < 3; i++ {
		if err := deployResources(project); err == nil {
			break
		} else if i == 2 {
			return fmt.Errorf("failed to deploy resources: %v", err)
		}
		log.Printf("Sleeping for %v and retrying in case failure was due to concurrent IAM policy update", deploymentRetryWaitTime)
		time.Sleep(deploymentRetryWaitTime)
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
	if err := deployAudit(project, config.ProjectForAuditLogs(project)); err != nil {
		return fmt.Errorf("failed to deploy audit resources: %v", err)
	}

	if err := deployGKEWorkloads(project); err != nil {
		return fmt.Errorf("failed to deploy GKE workloads: %v", err)
	}

	// Only remove owner account if there is an organization to ensure the project has an administrator.
	if config.Overall.OrganizationID != "" {
		if err := removeOwnerUser(project); err != nil {
			log.Printf("failed to remove owner user (might have already been removed): %v", err)
		}
	}
	return nil
}

// grantDeploymentManagerAccess grants the necessary permissions to the DM service account to perform its actions.
// Note: we don't revoke deployment manager's access because permissions can take up to 7 minutes
// to propagate through the system, which can cause permission denied issues when doing updates.
// This is not a problem on initial deployment since no resources have been created.
// DM is HIPAA compliant, so it's ok to leave its access.
// See https://cloud.google.com/iam/docs/granting-changing-revoking-access.
func grantDeploymentManagerAccess(project *Project) error {
	pnum := project.GeneratedFields.ProjectNumber
	if pnum == "" {
		return fmt.Errorf("project number not set in generated fields %+v", project.GeneratedFields)
	}
	serviceAcct := fmt.Sprintf("serviceAccount:%s@cloudservices.gserviceaccount.com", pnum)

	// TODO: account for this in the rule generator.
	for _, role := range deploymentManagerRoles {
		cmd := exec.Command(
			"gcloud", "projects", "add-iam-policy-binding", project.ID,
			"--role", role,
			"--member", serviceAcct,
			"--project", project.ID,
		)
		if err := cmdRun(cmd); err != nil {
			return fmt.Errorf("failed to grant role %q to DM service account %q: %v", role, serviceAcct, err)
		}
	}
	return nil
}

func getLogSinkServiceAccount(project *Project) (string, error) {
	cmd := exec.Command("gcloud", "logging", "sinks", "describe", project.BQLogSink.Name(), "--format", "json", "--project", project.ID)

	out, err := cmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to query log sink service account from gcloud: %v", err)
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

	importSet := make(map[string]bool)

	for _, pair := range pairs {
		var typ string
		if typer, ok := pair.parsed.(deploymentManagerTyper); ok {
			typ = typer.DeploymentManagerType()
		} else if pather, ok := pair.parsed.(deploymentManagerPather); ok {
			var err error
			typ, err = filepath.Abs(pather.TemplatePath())
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for %q: %v", pather.TemplatePath(), err)
			}
			if !importSet[typ] {
				deployment.Imports = append(deployment.Imports, &deploymentmanager.Import{Path: typ})
				importSet[typ] = true
			}
		} else {
			return nil, fmt.Errorf("failed to get type of %+v", pair.parsed)
		}

		merged, err := pair.MergedPropertiesMap()
		if err != nil {
			return nil, fmt.Errorf("failed to merge raw map with parsed: %v", err)
		}

		res := &deploymentmanager.Resource{
			Name:       pair.parsed.Name(),
			Type:       typ,
			Properties: merged,
		}

		if dr, ok := pair.parsed.(depender); ok && len(dr.Dependencies()) > 0 {
			res.Metadata = &deploymentmanager.Metadata{DependsOn: dr.Dependencies()}
		}

		deployment.Resources = append(deployment.Resources, res)
	}

	return deployment, nil
}

func removeOwnerUser(project *Project) error {
	cmd := exec.Command("gcloud", "config", "get-value", "account", "--format", "json", "--project", project.ID)
	out, err := cmdOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to get currently authenticated user: %v", err)
	}
	var member string
	if err := json.Unmarshal(out, &member); err != nil {
		return fmt.Errorf("failed to unmarshal current user: %v", err)
	}
	role := "roles/owner"
	member = "user:" + member

	// TODO: check user specified bindings in case user wants the binding left
	has, err := hasBinding(project, role, member)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}

	cmd = exec.Command(
		"gcloud", "projects", "remove-iam-policy-binding", project.ID,
		"--member", member, "--role", role, "--project", project.ID)
	return cmdRun(cmd)
}

func hasBinding(project *Project, role string, member string) (has bool, err error) {
	cmd := exec.Command(
		"gcloud", "projects", "get-iam-policy", project.ID,
		"--project", project.ID,
		"--format", "json",
	)
	out, err := cmdOutput(cmd)
	if err != nil {
		return false, fmt.Errorf("failed to get iam policy bindings: %v", err)
	}
	log.Printf("Looking for role %q, member %q in:\n%v", role, member, string(out))

	type policy struct {
		Bindings []Binding `json:"bindings"`
	}
	p := new(policy)
	if err := json.Unmarshal(out, p); err != nil {
		return false, fmt.Errorf("failed to unmarshal get-iam-policy output: %v", err)
	}
	for _, b := range p.Bindings {
		if b.Role == role {
			for _, m := range b.Members {
				if m == member {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
