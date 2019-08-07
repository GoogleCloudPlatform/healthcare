// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"errors"
	"fmt"
)

// BigqueryDataset represents a bigquery dataset.
type BigqueryDataset struct {
	BigqueryDatasetProperties `json:"properties"`
	raw                       json.RawMessage
}

// BigqueryDatasetProperties represents a partial CFT dataset implementation.
type BigqueryDatasetProperties struct {
	BigqueryDatasetName string    `json:"name"`
	Location            string    `json:"location"`
	Accesses            []*Access `json:"access,omitempty"`
	SetDefaultOwner     bool      `json:"setDefaultOwner,omitempty"`
}

// Access defines a dataset access. Only one non-role field should be set.
type Access struct {
	Role         string `json:"role"`
	UserByEmail  string `json:"userByEmail,omitempty"`
	GroupByEmail string `json:"groupByEmail,omitempty"`

	// Unsupported roles.
	SpecialGroup string      `json:"specialGroup,omitempty"`
	View         interface{} `json:"view,omitempty"`
}

// Init initializes a new dataset with the given project.
func (d *BigqueryDataset) Init() error {
	if d.Name() == "" {
		return errors.New("name must be set")
	}
	if d.Location == "" {
		return errors.New("location must be set")
	}
	if d.SetDefaultOwner {
		return errors.New("setDefaultOwner must not be true")
	}
	return nil
}

// Name returns the name of this dataset.
func (d *BigqueryDataset) Name() string {
	return d.BigqueryDatasetName
}

// TemplatePath returns the name of the template to use for this dataset.
func (d *BigqueryDataset) TemplatePath() string {
	return "deploy/config/templates/bigquery/bigquery_dataset.py"
}

// aliasBQDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasBigqueryDataset BigqueryDataset

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (d *BigqueryDataset) UnmarshalJSON(data []byte) error {
	var alias aliasBigqueryDataset
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*d = BigqueryDataset(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (d *BigqueryDataset) MarshalJSON() ([]byte, error) {
	return interfacePair{d.raw, aliasBigqueryDataset(*d)}.MarshalJSON()
}
