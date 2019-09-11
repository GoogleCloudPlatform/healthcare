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

// ComputeImage represents a Terraform GCE compute image.
type ComputeImage struct {
	Name    string `json:"name"`
	Project string `json:"project"`

	// TODO: add documentation on this var as well in all raws.
	raw json.RawMessage
}

// Init initializes the resource.
func (i *ComputeImage) Init(projectID string) error {
	if i.Name == "" {
		return errors.New("name must be set")
	}
	if i.Project != "" {
		return fmt.Errorf("project must not be set: %q", i.Project)
	}
	i.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (i *ComputeImage) ID() string {
	return i.Name
}

// ResourceType returns the resource terraform provider type.
func (i *ComputeImage) ResourceType() string {
	return "google_compute_image"
}

// ImportID returns the ID to use for terraform imports.
func (i *ComputeImage) ImportID() (string, error) {
	return fmt.Sprintf("%s/%s", i.Project, i.Name), nil
}

// aliasStorageBucket is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasComputeImage ComputeImage

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (i *ComputeImage) UnmarshalJSON(data []byte) error {
	var alias aliasComputeImage
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*i = ComputeImage(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (i *ComputeImage) MarshalJSON() ([]byte, error) {
	return interfacePair{i.raw, aliasComputeImage(*i)}.MarshalJSON()
}

// ComputeInstance represents a Terraform GCE compute instance.
type ComputeInstance struct {
	Name    string `json:"name"`
	Project string `json:"project"`
	Zone    string `json:"zone"`

	raw json.RawMessage
}

// Init initializes the resource.
func (i *ComputeInstance) Init(projectID string) error {
	if i.Name == "" {
		return errors.New("name must be set")
	}
	if i.Project != "" {
		return fmt.Errorf("project must not be set: %q", i.Project)
	}
	i.Project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (i *ComputeInstance) ID() string {
	return i.Name
}

// ResourceType returns the resource terraform provider type.
func (i *ComputeInstance) ResourceType() string {
	return "google_compute_instance"
}

// ImportID returns the ID to use for terraform imports.
func (i *ComputeInstance) ImportID() (string, error) {
	return fmt.Sprintf("%s/%s/%s", i.Project, i.Zone, i.Name), nil
}

// aliasStorageBucket is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasComputeInstance ComputeInstance

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (i *ComputeInstance) UnmarshalJSON(data []byte) error {
	var alias aliasComputeInstance
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*i = ComputeInstance(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (i *ComputeInstance) MarshalJSON() ([]byte, error) {
	return interfacePair{i.raw, aliasComputeInstance(*i)}.MarshalJSON()
}
