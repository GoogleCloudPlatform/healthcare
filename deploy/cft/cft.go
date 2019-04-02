// Package cft provides utilities to deploy CFT resources.
package cft

import (
	"fmt"
	"path/filepath"

	"github.com/ghodss/yaml"
)

// Config represents a (partial) representation of a projects YAML file.
// Only the required fields are present. See project_config.yaml.schema for details.
type Config struct {
	Projects []*Project `json:"projects"`
}

// Project defines a single project's configuration.
type Project struct {
	ID                  string   `json:"project_id"`
	OwnersGroup         string   `json:"owners_group"`
	DataReadWriteGroups []string `json:"data_readwrite_groups"`
	DataReadOnlyGroups  []string `json:"data_readonly_groups"`
	Resources           []struct {
		BigqueryDataset interface{} `json:"bigquery_dataset"`
		GCSBucket       interface{} `json:"gcs_bucket"`
		GKECluster      interface{} `json:"gke_cluster"`
		GKEWorkload     interface{} `json:"gke_workload"`
	} `json:"resources"`
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
	deployment, err := getDeployment(project)
	if err != nil {
		return err
	}
	return createOrUpdateDeployment(project.ID, deployment)
}

func getDeployment(project *Project) (*Deployment, error) {
	deployment := &Deployment{}

	for _, r := range project.Resources {
		// typically only one of these will have a raw that is not nil
		initialPairs := []resourcePair{
			{&BigqueryDataset{}, r.BigqueryDataset},
			{NewGCSBucket(), r.GCSBucket},
			{&DefaultResource{templatePath: "deploy/cft/templates/gke.py"}, r.GKECluster},
		}

		for _, pair := range initialPairs {
			if pair.raw == nil {
				continue
			}

			if err := unmarshal(pair.raw, pair.parsed); err != nil {
				return nil, fmt.Errorf("failed to unmarshal %q: %v", pair.parsed.TemplatePath(), err)
			}

			resources, importSet, err := getDeploymentResourcesAndImports(pair, project)
			if err != nil {
				return nil, fmt.Errorf("failed to get resource and imports for %q: %v", pair.parsed.Name(), err)
			}
			deployment.Resources = append(deployment.Resources, resources...)

			for imp := range importSet {
				deployment.Imports = append(deployment.Imports, &Import{Path: imp})
			}
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

	if err := pair.parsed.Init(project); err != nil {
		return nil, nil, fmt.Errorf("failed to init %q: %v", pair.parsed.TemplatePath(), err)
	}

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
		depPair := resourcePair{raw: make(map[string]interface{}), parsed: dep}
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

// unmarshal converts one YAML supported type into another.
// Note: both in and out must support YAML marshalling.
// See yaml.Unmarshal details on supported types.
func unmarshal(in interface{}, out interface{}) error {
	b, err := yaml.Marshal(in)
	if err != nil {
		return fmt.Errorf("failed to marshal %v: %v", in, err)
	}

	if err := yaml.Unmarshal(b, out); err != nil {
		return fmt.Errorf("failed to unmarshal %v: %v", string(b), err)
	}
	return nil
}
