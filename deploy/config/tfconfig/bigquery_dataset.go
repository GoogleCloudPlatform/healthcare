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

package tfconfig

import (
	"encoding/json"
	"errors"
	"fmt"
)

// BigqueryDataset represents a bigquery dataset.
type BigqueryDataset struct {
	DatasetID string `json:"dataset_id"`
	Project   string `json:"project"`
	Location  string `json:"location"`

	// Note: accesses are authoritative, meaning they will overwrite any existing access.
	// We can support non-authoritative access after https://github.com/terraform-providers/terraform-provider-google/issues/3990 is fixed.
	// TODO: set access on the dataset by default so it overrides the default which
	// grants the running user owners permission.
	Accesses []*Access `json:"access,omitempty"`

	raw json.RawMessage
}

// Access defines a dataset access. Only one non-role field should be set.
type Access struct {
	Role         string `json:"role"`
	UserByEmail  string `json:"user_by_email,omitempty"`
	GroupByEmail string `json:"group_by_email,omitempty"`

	// Unsupported roles.
	SpecialGroup string      `json:"special_group,omitempty"`
	View         interface{} `json:"view,omitempty"`
}

// Init initializes the resource.
func (d *BigqueryDataset) Init(projectID string) error {
	if d.DatasetID == "" {
		return errors.New("name must be set")
	}
	if d.Location == "" {
		return errors.New("location must be set")
	}
	d.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (d *BigqueryDataset) ID() string {
	return d.DatasetID
}

// ResourceType returns the resource terraform provider type.
func (d *BigqueryDataset) ResourceType() string {
	return "google_bigquery_dataset"
}

// ImportID returns the ID to use for terraform imports.
func (d *BigqueryDataset) ImportID() string {
	return fmt.Sprintf("%s:%s", d.Project, d.ID())
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
