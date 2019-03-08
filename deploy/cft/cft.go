// Package cft provides utilities to deploy CFT resources.
package cft

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
}
