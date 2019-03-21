package cft

import (
	"errors"
)

// BigqueryDataset represents a bigquery dataset.
type BigqueryDataset struct {
	BigqueryDatasetProperties `yaml:"properties"`
}

// BigqueryDatasetProperties represents a partial CFT dataset implementation.
type BigqueryDatasetProperties struct {
	BigqueryDatasetName string   `yaml:"name"`
	Accesses            []access `yaml:"access"`
	SetDefaultOwner     bool     `yaml:"setDefaultOwner"`
}

type access struct {
	Role         string `yaml:"role"`
	UserByEmail  string `yaml:"userByEmail,omitempty"`
	GroupByEmail string `yaml:"groupByEmail,omitempty"`
	SpecialGroup string `yaml:"specialGroup,omitempty"`

	// View is allowed but not monitored. Parse it into a generic map.
	View map[string]interface{} `yaml:"view,omitempty"`
}

// Init initializes a new dataset with the given project.
func (d *BigqueryDataset) Init(project *Project) error {
	if d.Name() == "" {
		return errors.New("name must be set")
	}
	if d.SetDefaultOwner {
		return errors.New("setDefaultOwner must not be true")
	}

	// Note: duplicate accesses are de-duplicated by deployment manager.
	roleAndGroups := []struct {
		Role   string
		Groups []string
	}{
		{"OWNER", []string{project.OwnersGroup}},
		{"WRITER", project.DataReadWriteGroups},
		{"READER", project.DataReadOnlyGroups},
	}

	for _, rg := range roleAndGroups {
		for _, g := range rg.Groups {
			d.Accesses = append(d.Accesses, access{
				Role:         rg.Role,
				GroupByEmail: g,
			})
		}
	}

	return nil
}

// Name returns the name of this dataset.
func (d *BigqueryDataset) Name() string {
	return d.BigqueryDatasetName
}

// TemplatePath returns the name of the template to use for this dataset.
func (d *BigqueryDataset) TemplatePath() string {
	return "bigquery_dataset.py"
}
