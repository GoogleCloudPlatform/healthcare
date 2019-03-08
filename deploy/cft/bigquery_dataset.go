package cft

import (
	"errors"
)

// BigqueryDataset represents a bigquery dataset.
type BigqueryDataset struct {
	BigQueryDatasetCFT `yaml:"cft"`
}

// BigQueryDatasetCFT represents a partial CFT dataset implementation.
type BigQueryDatasetCFT struct {
	Name            string   `yaml:"name"`
	Accesses        []access `yaml:"access"`
	SetDefaultOwner bool     `yaml:"setDefaultOwner"`
}

type access struct {
	Role         string `yaml:"role"`
	UserByEmail  string `yaml:"userByEmail,omitempty"`
	GroupByEmail string `yaml:"groupByEmail,omitempty"`
	SpecialGroup string `yaml:"specialGroup,omitempty"`

	// View is allowed but not monitored. Parse it into a generic map.
	View map[string]interface{} `yaml:"view,omitempty"`
}

// NewBigqueryDataset initializes a new dataset.
func NewBigqueryDataset(project *Project, opts ...func(*BigqueryDataset) error) (*BigqueryDataset, error) {
	d := &BigqueryDataset{}

	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}

	if d.Name == "" {
		return nil, errors.New("name must be set")
	}
	if d.SetDefaultOwner {
		return nil, errors.New("setDefaultOwner must not be true")
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

	return d, nil
}

// ResourceName returns the name of this dataset.
func (d *BigqueryDataset) ResourceName() string {
	return d.Name
}
