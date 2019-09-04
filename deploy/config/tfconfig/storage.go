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

	IAMMembers []*StorageIAMMember `json:"_iam_members"`

	raw json.RawMessage
}

type versioning struct {
	// Use pointer to differentiate between zero value and intentionally being set to false.
	Enabled *bool `json:"enabled"`
}

// Init initializes the resource.
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

// ID returns the resource unique identifier.
func (b *StorageBucket) ID() string {
	return b.Name
}

// ResourceType returns the resource terraform provider type.
func (b *StorageBucket) ResourceType() string {
	return "google_storage_bucket"
}

// DependentResources returns the child resources of this resource.
func (b *StorageBucket) DependentResources() []Resource {
	if len(b.IAMMembers) == 0 {
		return nil
	}

	forEach := make(map[string]*StorageIAMMember)
	for _, m := range b.IAMMembers {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}
	return []Resource{&StorageIAMMember{
		ForEach: forEach,
		Bucket:  fmt.Sprintf("${google_storage_bucket.%s.name}", b.Name),
		Role:    "${each.value.role}",
		Member:  "${each.value.member}",
		id:      b.Name,
	}}
}

// ImportID returns the ID to use for terraform imports.
func (b *StorageBucket) ImportID() string {
	return fmt.Sprintf("%s/%s", b.Project, b.ID())
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

// StorageIAMMember represents a Terraform GCS bucket IAM member.
type StorageIAMMember struct {
	Role   string `json:"role"`
	Member string `json:"member"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*StorageIAMMember `json:"for_each,omitempty"`

	// Bucket should be written as a terraform reference to a bucket name so that it is created after the bucket.
	// e.g. ${google_storage_bucket.foo_bucket.name}
	Bucket string `json:"bucket,omitempty"`

	// id should be the bucket's literal name.
	id string
}

// Init initializes the resource.
func (m *StorageIAMMember) Init(string) error {
	return nil
}

// ID returns the unique identifier.
func (m *StorageIAMMember) ID() string {
	return m.id
}

// ResourceType returns the terraform provider type.
func (m *StorageIAMMember) ResourceType() string {
	return "google_storage_bucket_iam_member"
}
