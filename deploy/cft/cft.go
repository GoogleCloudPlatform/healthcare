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
	} `yaml:"resources"`
}

// parsedResource is an interface that must be implemented by all concrete resource implementations.
type parsedResource interface {
	Init(*Project) error
	Name() string
}

// Deploy deploys the CFT resources in the project.
func Deploy(project *Project) error {
	templateToResourcePairs, err := getTemplateToResourcePairs(project)
	if err != nil {
		return err
	}

	deployment := &Deployment{}

	for template, pairs := range templateToResourcePairs {
		path, err := getCFTTemplatePath(template)
		if err != nil {
			return err
		}
		deployment.Imports = append(deployment.Imports, Import{Path: path})
		for _, pair := range pairs {
			if err := pair.parsed.Init(project); err != nil {
				return fmt.Errorf("failed to init %q: %v", pair.parsed.Name(), err)
			}
			merged, err := pair.MergedPropertiesMap()
			if err != nil {
				return fmt.Errorf("failed to merge raw map with parsed: %v", err)
			}
			deployment.Resources = append(deployment.Resources, Resource{
				Name:       pair.parsed.Name(),
				Type:       path,
				Properties: merged,
			})
		}
	}

	return createOrUpdateDeployment(project.ID, deployment)
}

// getTemplateToResourcePairs gets the map of a template file name to its resource pairs for all resources in the project.
// TODO: support additional resources.
func getTemplateToResourcePairs(p *Project) (map[string][]resourcePair, error) {
	res := make(map[string][]resourcePair)

	for _, r := range p.Resources {
		if r.BigqueryDataset != nil {
			d := new(BigqueryDataset)
			if err := unmarshal(r.BigqueryDataset, d); err != nil {
				return nil, err
			}
			t := "bigquery_dataset.py"
			res[t] = append(res[t], resourcePair{parsed: d, raw: r.BigqueryDataset})
		}
	}

	return res, nil
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
