package cft

import (
	"errors"
)

// BigqueryDataset represents a bigquery dataset.
type BigqueryDataset struct {
	BigqueryDatasetProperties `json:"properties"`
}

// BigqueryDatasetProperties represents a partial CFT dataset implementation.
type BigqueryDatasetProperties struct {
	BigqueryDatasetName string   `json:"name"`
	Accesses            []Access `json:"access"`
	SetDefaultOwner     bool     `json:"setDefaultOwner"`
}

// Access defines a dataset access. Only one non-role field should be set.
type Access struct {
	Role         string `json:"role"`
	UserByEmail  string `json:"userByEmail,omitempty"`
	GroupByEmail string `json:"groupByEmail,omitempty"`
	SpecialGroup string `json:"specialGroup,omitempty"`

	// View is allowed but not monitored. Parse it into a generic map.
	View map[string]interface{} `json:"view,omitempty"`
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
			d.Accesses = append(d.Accesses, Access{
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
	return "deploy/cft/templates/bigquery_dataset.py"
}
