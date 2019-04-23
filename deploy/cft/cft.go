// Package cft provides utilities to deploy CFT resources.
package cft

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
)

// Config represents a (partial) representation of a projects YAML file.
// Only the required fields are present. See project_config.yaml.schema for details.
// TODO: move config to its own package
type Config struct {
	Projects []*Project `json:"projects"`
}

// Project defines a single project's configuration.
type Project struct {
	ID                  string   `json:"project_id"`
	OwnersGroup         string   `json:"owners_group"`
	DataReadWriteGroups []string `json:"data_readwrite_groups"`
	DataReadOnlyGroups  []string `json:"data_readonly_groups"`

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

	AuditLogs struct {
		LogsGCSBucket struct {
			Name string `json:"name"`
		} `json:"logs_gcs_bucket"`
	} `json:"audit_logs"`
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
	Parsed DefaultResource `json:"-"`
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

// Init initializes a project.
func (p *Project) Init() error {
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
		appendDefaultResPair(res.GCEInstancePair.Raw, &res.GCEInstancePair.Parsed, "deploy/cft/templates/instance.py")
		appendPair(res.GCSBucketPair.Raw, &res.GCSBucketPair.Parsed)
		appendPair(res.GKEClusterPair.Raw, &res.GKEClusterPair.Parsed)
		appendPair(res.PubsubPair.Raw, &res.PubsubPair.Parsed)
	}
	return pairs
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
	return createOrUpdateDeployment(project.ID, deployment)
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

		for imp := range importSet {
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
