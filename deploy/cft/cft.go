// Package cft provides utilities to deploy CFT resources.
package cft

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
)

// Config represents a (partial) representation of a projects YAML file.
// Only the required fields are present. See project_config.yaml.schema for details.
// TODO: move config to its own package
type Config struct {
	Overall struct {
		OrganizationID string   `json:"organization_id"`
		FolderID       string   `json:"folder_id"`
		AllowedAPIs    []string `json:"allowed_apis"`
	} `json:"overall"`
	AuditLogsProject *Project `json:"audit_logs_project"`
	Forseti          *struct {
		Project         *Project `json:"project"`
		GeneratedFields struct {
			ServiceAccount string `json:"service_account"`
			ServerBucket   string `json:"server_bucket"`
		} `json:"generated_fields"`
	} `json:"forseti"`
	Projects []*Project `json:"projects"`
}

// Project defines a single project's configuration.
type Project struct {
	ID                  string   `json:"project_id"`
	OwnersGroup         string   `json:"owners_group"`
	AuditorsGroup       string   `json:"auditors_group"`
	DataReadWriteGroups []string `json:"data_readwrite_groups"`
	DataReadOnlyGroups  []string `json:"data_readonly_groups"`
	EnabledAPIs         []string `json:"enabled_apis"`

	// Note: typically only one resource in the struct is set at one time.
	// Go does not have the concept of "one-of", so despite only one of these resources
	// typically being non-empty, we still check them all. The one-of check is done by the schema.
	Resources []*struct {
		// The following structs are embedded so the json parser skips directly to their fields.
		BigqueryDatasetPair
		FirewallPair
		GCEInstancePair
		GCSBucketPair
		GKEClusterPair
		PubsubPair

		// TODO: make this behave more like standard deployment manager resources
		GKEWorkload json.RawMessage `json:"gke_workload"`
	} `json:"resources"`

	AuditLogs *struct {
		LogsGCSBucket struct {
			Name     string `json:"name"`
			Location string `json:"location"`
		} `json:"logs_gcs_bucket"`

		LogsBigqueryDataset struct {
			Name     string `json:"name"`
			Location string `json:"location"`
		} `json:"logs_bigquery_dataset"`
	} `json:"audit_logs"`

	GeneratedFields struct {
		LogSinkServiceAccount string `json:"log_sink_service_account"`
		GCEInstanceInfo       []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"gce_instance_info"`
	} `json:"generated_fields"`
}

// BigqueryDatasetPair pairs a raw dataset with its parsed version.
type BigqueryDatasetPair struct {
	Raw    json.RawMessage `json:"bigquery_dataset"`
	Parsed BigqueryDataset `json:"-"`
}

// FirewallPair pairs a raw firewall with its parsed version.
type FirewallPair struct {
	Raw    json.RawMessage `json:"firewall"`
	Parsed DefaultResource `json:"-"`
}

// GCEInstancePair pairs a raw instance with its parsed version.
type GCEInstancePair struct {
	Raw    json.RawMessage `json:"gce_instance"`
	Parsed GCEInstance     `json:"-"`
}

// GCSBucketPair pairs a raw bucket with its parsed version.
type GCSBucketPair struct {
	Raw    json.RawMessage `json:"gcs_bucket"`
	Parsed GCSBucket       `json:"-"`
}

// GKEClusterPair pairs a raw cluster with its parsed version.
type GKEClusterPair struct {
	Raw    json.RawMessage `json:"gke_cluster"`
	Parsed GKECluster      `json:"-"`
}

// PubsubPair pairs a raw pubsub with its parsed version.
type PubsubPair struct {
	Raw    json.RawMessage `json:"pubsub"`
	Parsed Pubsub          `json:"-"`
}

// Init initializes the config and all its projects.
func (c *Config) Init() error {
	for _, p := range c.Projects {
		if err := p.Init(); err != nil {
			return fmt.Errorf("failed to init project %q: %v", p.ID, err)
		}
	}
	return nil
}

// AuditLogsProjectID is a helper function to get the audit logs project ID for the given project.
// If a remote audit logs project exists, return it will host all other projects' audit logs.
// Else, each project will locally host their own.
func (c *Config) AuditLogsProjectID(p *Project) string {
	if c.AuditLogsProject != nil {
		return c.AuditLogsProject.ID
	}
	return p.ID
}

// Init initializes a project and all its resources.
func (p *Project) Init() error {
	if p.AuditLogs.LogsBigqueryDataset.Name == "" {
		p.AuditLogs.LogsBigqueryDataset.Name = "audit_logs"
	}
	if p.AuditLogs.LogsGCSBucket.Name == "" {
		p.AuditLogs.LogsGCSBucket.Name = p.ID + "-logs"
	}
	for _, pair := range p.resourcePairs() {
		if err := json.Unmarshal(pair.raw, pair.parsed); err != nil {
			return err
		}
		if err := pair.parsed.Init(p); err != nil {
			return err
		}
	}
	return nil
}

func (p *Project) resourcePairs() []resourcePair {
	var pairs []resourcePair
	appendPair := func(raw json.RawMessage, parsed parsedResource) {
		if len(raw) > 0 {
			pairs = append(pairs, resourcePair{raw, parsed})
		}
	}
	appendDefaultResPair := func(raw json.RawMessage, parsed *DefaultResource, path string) {
		parsed.templatePath = path
		appendPair(raw, parsed)
	}
	for _, res := range p.Resources {
		appendPair(res.BigqueryDatasetPair.Raw, &res.BigqueryDatasetPair.Parsed)
		appendDefaultResPair(res.FirewallPair.Raw, &res.FirewallPair.Parsed, "deploy/cft/templates/firewall.py")
		appendPair(res.GCEInstancePair.Raw, &res.GCEInstancePair.Parsed)
		appendPair(res.GCSBucketPair.Raw, &res.GCSBucketPair.Parsed)
		appendPair(res.GKEClusterPair.Raw, &res.GKEClusterPair.Parsed)
		appendPair(res.PubsubPair.Raw, &res.PubsubPair.Parsed)
	}
	return pairs
}

// DataResources represents all data holding resources in the project.
type DataResources struct {
	BigqueryDatasets []*BigqueryDataset
	GCSBuckets       []*GCSBucket
	GCEInstances     []*GCEInstance
}

// DataResources gets all data holding resources in this project.
func (p *Project) DataResources() *DataResources {
	rs := &DataResources{}
	for _, r := range p.Resources {
		switch {
		case len(r.BigqueryDatasetPair.Raw) > 0:
			rs.BigqueryDatasets = append(rs.BigqueryDatasets, &r.BigqueryDatasetPair.Parsed)
		case len(r.GCSBucketPair.Raw) > 0:
			rs.GCSBuckets = append(rs.GCSBuckets, &r.GCSBucketPair.Parsed)
		case len(r.GCEInstancePair.Raw) > 0:
			rs.GCEInstances = append(rs.GCEInstances, &r.GCEInstancePair.Parsed)
		}
	}
	return rs
}

// InstanceID returns the ID of the instance with the given name.
func (p *Project) InstanceID(name string) (string, error) {
	for _, info := range p.GeneratedFields.GCEInstanceInfo {
		if info.Name == name {
			return info.ID, nil
		}
	}
	return "", fmt.Errorf("info for instance %q not found in generated_fields", name)
}

// parsedResource is an interface that must be implemented by all concrete resource implementations.
type parsedResource interface {
	Init(*Project) error
	Name() string
	TemplatePath() string
}

// depender is the interface that defines a method to get dependent resources.
type depender interface {
	DependentResources(*Project) ([]parsedResource, error)
}

// Deploy deploys the CFT resources in the project.
func Deploy(project *Project) error {
	pairs := project.resourcePairs()
	if len(pairs) == 0 {
		log.Println("No resources to deploy.")
		return nil
	}

	deployment, err := getDeployment(project, pairs)
	if err != nil {
		return err
	}
	if err := createOrUpdateDeployment(project.ID, deployment); err != nil {
		return fmt.Errorf("failed to deploy deployment manager resources: %v", err)
	}

	if err := deployGKEWorkloads(project); err != nil {
		return fmt.Errorf("failed to deploy GKE workloads: %v", err)
	}

	return nil
}

func getDeployment(project *Project, pairs []resourcePair) (*Deployment, error) {
	deployment := &Deployment{}

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
				deployment.Imports = append(deployment.Imports, &Import{Path: imp})
			}
			allImports[imp] = true
		}
	}

	return deployment, nil
}

// getDeploymentResourcesAndImports gets the deploment resources and imports for the given resource pair.
// It also recursively adds the resoures and imports of any dependent resources.
func getDeploymentResourcesAndImports(pair resourcePair, project *Project) (resources []*Resource, importSet map[string]bool, err error) {
	importSet = make(map[string]bool)

	templatePath, err := filepath.Abs(pair.parsed.TemplatePath())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get absolute path for %q: %v", pair.parsed.TemplatePath(), err)
	}
	importSet[templatePath] = true

	merged, err := pair.MergedPropertiesMap()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to merge raw map with parsed: %v", err)
	}

	resources = []*Resource{{
		Name:       pair.parsed.Name(),
		Type:       templatePath,
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
				res.Metadata = &Metadata{}
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
