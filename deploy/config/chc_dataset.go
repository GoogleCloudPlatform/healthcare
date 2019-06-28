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
	"errors"
)

// CHCDataset represents a CHC dataset.
type CHCDataset struct {
	CHCDatasetProperties `json:"properties"`
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
