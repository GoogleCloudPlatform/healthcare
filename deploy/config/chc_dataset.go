/*
 * Copyright 2019 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"encoding/json"
	"errors"
	"fmt"
)

// CHCDataset represents a CHC dataset.
type CHCDataset struct {
	CHCDatasetProperties `json:"properties"`
	raw                  json.RawMessage
}

// CHCDatasetProperties represents a partial CFT dataset implementation.
type CHCDatasetProperties struct {
	CHCDatasetID string `json:"datasetId"`
}

// Init initializes a new dataset with the given project.
func (d *CHCDataset) Init(project *Project) error {
	if d.Name() == "" {
		return errors.New("name must be set")
	}
	return nil
}

// Name returns the name of this dataset.
func (d *CHCDataset) Name() string {
	return d.CHCDatasetID
}

// TemplatePath returns the name of the template to use for this dataset.
func (d *CHCDataset) TemplatePath() string {
	return "deploy/config/templates/chc_resource/chc_dataset.py"
}

// aliasCHCDataset is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasCHCDataset CHCDataset

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (d *CHCDataset) UnmarshalJSON(data []byte) error {
	var alias aliasCHCDataset
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*d = CHCDataset(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (d *CHCDataset) MarshalJSON() ([]byte, error) {
	return interfacePair{d.raw, aliasCHCDataset(*d)}.MarshalJSON()
}
