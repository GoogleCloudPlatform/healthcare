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

// StorageBucket represents a Terraform GCS bucket.
type StorageBucket struct {
	Name       string     `json:"name"`
	Project    string     `json:"project"`
	Location   string     `json:"location"`
	Versioning versioning `json:"versioning,omitempty"`
	raw        json.RawMessage
}

type versioning struct {
	// Use pointer to differentiate between zero value and intentionally being set to false.
	Enabled *bool `json:"enabled"`
}

// Init initializes the bucket.
func (b *StorageBucket) Init(projectID string) error {
	if b.Name == "" {
		return errors.New("name must be set")
	}
	if b.Project != "" {
		return fmt.Errorf("project must not be set: %q", b.Project)
	}
	b.Project = projectID

	if b.Versioning.Enabled != nil && !*b.Versioning.Enabled {
		return errors.New("versioning must not be disabled")
	}
	t := true
	b.Versioning.Enabled = &t
	return nil
}

// TerraformResourceName returns the Google provider terraform resource.
func (b *StorageBucket) TerraformResourceName() string {
	return "google_storage_bucket"
}

// ID returns the unique identifier of this bucket.
func (b *StorageBucket) ID() string {
	return b.Name
}

// aliasStorageBucket is used to prevent infinite recursion when dealing with json marshaling.
// https://stackoverflow.com/q/52433467
type aliasStorageBucket StorageBucket

// UnmarshalJSON provides a custom JSON unmarshaller.
// It is used to store the original (raw) user JSON definition,
// which can have more fields than what is defined in this struct.
func (b *StorageBucket) UnmarshalJSON(data []byte) error {
	var alias aliasStorageBucket
	if err := unmarshalJSONMany(data, &alias, &alias.raw); err != nil {
		return fmt.Errorf("failed to unmarshal to parsed alias: %v", err)
	}
	*b = StorageBucket(alias)
	return nil
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to merge the original (raw) user JSON definition with the struct.
func (b *StorageBucket) MarshalJSON() ([]byte, error) {
	return interfacePair{b.raw, aliasStorageBucket(*b)}.MarshalJSON()
}
