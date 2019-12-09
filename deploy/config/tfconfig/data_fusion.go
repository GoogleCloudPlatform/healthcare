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

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
)

// DataFusionInstance represents a Terraform Data Fusion Instance.
type DataFusionInstance struct {
	Name    string `json:"name"`
	Project string `json:"project"`
	Type    string `json:"type"`
	Region  string `json:"region"`

	Provider                    string `json:"provider,omitempty"`
	EnableStackdriverLogging    *bool  `json:"enable_stackdriver_logging,omitempty"`
	EnableStackdriverMonitoring *bool  `json:"enable_stackdriver_monitoring,omitempty"`

	raw json.RawMessage
}

// Init initializes the resource.
func (i *DataFusionInstance) Init(projectID string) error {
	if i.Name == "" {
		return errors.New("name must be set")
	}
	if i.Type == "" {
		return errors.New("type must be set")
	}
	if i.Region == "" {
		return errors.New("region must be set")
	}
	if i.Project != "" {
		return fmt.Errorf("project must be unset: %v", i.Project)
	}
	i.Project = projectID
	i.Provider = "google-beta"
	t := true
	if i.EnableStackdriverLogging == nil {
		i.EnableStackdriverLogging = &t
	}
	if i.EnableStackdriverMonitoring == nil {
		i.EnableStackdriverMonitoring = &t
	}
	return nil
}

// ID returns the resource unique identifier.
func (i *DataFusionInstance) ID() string {
	return i.Name
}

// ResourceType returns the resource terraform provider type.
func (i *DataFusionInstance) ResourceType() string {
	return "google_data_fusion_instance"
}

// ImportID returns the ID to use for terraform imports.
func (i *DataFusionInstance) ImportID(runner.Runner) (string, error) {
	return fmt.Sprintf("projects/%s/locations/%s/instances/%s", i.Project, i.Region, i.Name), nil
}

// aliasStorageBucket is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasDataFusionInstance DataFusionInstance

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (i *DataFusionInstance) UnmarshalJSON(data []byte) error {
	var alias aliasDataFusionInstance
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*i = DataFusionInstance(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (i *DataFusionInstance) MarshalJSON() ([]byte, error) {
	return interfacePair{i.raw, aliasDataFusionInstance(*i)}.MarshalJSON()
}
