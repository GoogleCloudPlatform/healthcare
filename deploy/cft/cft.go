// Package cft provides utilities to deploy CFT resources.
package cft

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const templatesRoot = "deploy/cft/templates"

// Config represents a (partial) representation of a projects YAML file.
// Only the required fields are present. See project_config.yaml.schema for details.
type Config struct {
	Projects []*Project `yaml:"projects"`
}

// Project defines a single project's configuration.
type Project struct {
	ID                  string   `yaml:"project_id"`
	OwnersGroup         string   `yaml:"owners_group"`
	DataReadWriteGroups []string `yaml:"data_readwrite_groups"`
	DataReadOnlyGroups  []string `yaml:"data_readonly_groups"`
	Resources           []struct {
		BigqueryDataset interface{} `yaml:"bigquery_dataset"`
		GCSBucket       interface{} `yaml:"gcs_bucket"`
		GKECluster      interface{} `yaml:"gke_cluster"`
	} `yaml:"resources"`
}

// parsedResource is an interface that must be implemented by all concrete resource implementations.
type parsedResource interface {
	Init(*Project) error
	Name() string
	TemplatePath() string
}

// Deploy deploys the CFT resources in the project.
func Deploy(project *Project) error {
	pairs, err := getResourcePairs(project)
	if err != nil {
		return err
	}

	deployment := &Deployment{}
	importSet := make(map[string]bool)

	for _, pair := range pairs {
		importSet[pair.parsed.TemplatePath()] = true

		merged, err := pair.MergedPropertiesMap()
		if err != nil {
			return fmt.Errorf("failed to merge raw map with parsed: %v", err)
		}
		deployment.Resources = append(deployment.Resources, Resource{
			Name:       pair.parsed.Name(),
			Type:       pair.parsed.TemplatePath(),
			Properties: merged,
		})
	}

	for imp := range importSet {
		path, err := getCFTTemplatePath(imp)
		if err != nil {
			return fmt.Errorf("failed to get template path for %q: %v", imp, err)
		}
		deployment.Imports = append(deployment.Imports, Import{Path: path})
	}

	return createOrUpdateDeployment(project.ID, deployment)
}

// getResourcePairs returns the resource pair for all resources in the project.
// TODO: support additional resources.
func getResourcePairs(project *Project) ([]resourcePair, error) {
	var pairs []resourcePair

	for _, r := range project.Resources {

		initialPairs := []resourcePair{
			{
				&BigqueryDataset{},
				r.BigqueryDataset,
			}, {
				&DefaultResource{templatePath: "gke.py"},
				r.GKECluster,
			},
		}

		for _, pair := range initialPairs {
			if pair.raw == nil {
				continue
			}

			if err := unmarshal(pair.raw, pair.parsed); err != nil {
				return nil, fmt.Errorf("failed to unmarshal %q: %v", pair.parsed.TemplatePath(), err)
			}

			if err := pair.parsed.Init(project); err != nil {
				return nil, fmt.Errorf("failed to init %q: %v", pair.parsed.Name(), err)
			}

			pairs = append(pairs, pair)
		}
	}
	return pairs, nil
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

// getCFTTemplatePath gets the absolute path for the given template file name.
func getCFTTemplatePath(name string) (string, error) {
	t, err := filepath.Abs(filepath.Join(templatesRoot, name))
	if err != nil {
		return "", fmt.Errorf("failed to get template path for %q: %v", name, err)
	}
	return t, nil
}
