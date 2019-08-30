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
	"fmt"
)

// ProjectIAMMembers represents multiple Terraform project IAM members.
// It is used to wrap and merge multiple IAM members into a single IAM member when being marshalled to JSON.
type ProjectIAMMembers struct {
	members []*ProjectIAMMember
	project string
}

// ProjectIAMMember represents a Terraform project IAM member.
type ProjectIAMMember struct {
	Role   string `json:"role"`
	Member string `json:"member"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]*ProjectIAMMember `json:"for_each,omitempty"`
	Project string                       `json:"project,omitempty"`
}

// Init initializes the resource.
func (ms *ProjectIAMMembers) Init(projectID string) error {
	ms.project = projectID
	return nil
}

// ID returns the resource unique identifier.
func (ms *ProjectIAMMembers) ID() string {
	return ms.project
}

// ResourceType returns the resource terraform provider type.
func (ms *ProjectIAMMembers) ResourceType() string {
	return "google_project_iam_member"
}

// MarshalJSON marshals the list of members into a single member.
// The single member will set a for_each block to expand to multiple iam members in the terraform call.
func (ms *ProjectIAMMembers) MarshalJSON() ([]byte, error) {
	forEach := make(map[string]*ProjectIAMMember)
	for _, m := range ms.members {
		key := fmt.Sprintf("%s %s", m.Role, m.Member)
		forEach[key] = m
	}

	return json.Marshal(&ProjectIAMMember{
		ForEach: forEach,
		Project: ms.project,
		Role:    "${each.value.role}",
		Member:  "${each.value.member}",
	})
}

// UnmarshalJSON unmarshals the bytes to a list of members.
func (ms *ProjectIAMMembers) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &ms.members)
}
