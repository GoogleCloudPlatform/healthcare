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
)

// ProjectResource represents a Terraform project resource.
// https://www.terraform.io/docs/providers/google/r/google_project.html
type ProjectResource struct {
	ProjectID      string `json:"project_id"`
	Name           string `json:"name"`
	OrgID          string `json:"org_id,omitempty"`
	FolderID       string `json:"folder_id,omitempty"`
	BillingAccount string `json:"billing_account"`
}

// Init initializes the resource.
func (p *ProjectResource) Init(projectID string) error {
	p.ProjectID = projectID
	// TODO: allow name to be set in config.
	if p.Name == "" {
		p.Name = projectID
	}
	return nil
}

// ID returns the resource unique identifier.
// It is hardcoded to return "project" as there is at most one of this resource in a deployment.
func (*ProjectResource) ID() string {
	return "project"
}

// ResourceType returns the resource terraform provider type.
func (p *ProjectResource) ResourceType() string {
	return "google_project"
}

// ImportID returns the ID to use for terraform imports.
func (p *ProjectResource) ImportID() (string, error) {
	return p.ProjectID, nil
}

// ProjectServices represents multiple Terraform project services.
// It is used to wrap and merge multiple services into a single service struct when being marshalled to JSON.
type ProjectServices struct {
	Services []*ProjectService
	project  string
}

// ProjectService represents a Terraform project service.
type ProjectService struct {
	Service string `json:"service"`

	// The following fields should not be set by users.

	// ForEach is used to let a single iam member expand to reference multiple iam members
	// through the use of terraform's for_each iterator.
	ForEach map[string]bool `json:"for_each,omitempty"`
	Project string          `json:"project,omitempty"`
}

// Init initializes the resource.
func (s *ProjectServices) Init(projectID string) error {
	s.project = projectID
	return nil
}

// ID returns the resource unique identifier.
// It is hardcoded to return "project" as there is at most one of this resource in a deployment.
func (*ProjectServices) ID() string {
	return "project"
}

// ResourceType returns the resource terraform provider type.
func (*ProjectServices) ResourceType() string {
	return "google_project_service"
}

// MarshalJSON marshals the list of members into a single member.
// The single member will set a for_each block to expand to multiple iam members in the terraform call.
func (s *ProjectServices) MarshalJSON() ([]byte, error) {
	forEach := make(map[string]bool)

	for _, svc := range s.Services {
		forEach[svc.Service] = true
	}

	return json.Marshal(&ProjectService{
		ForEach: forEach,
		Project: s.project,
		Service: "${each.key}",
	})
}

// UnmarshalJSON unmarshals the bytes to a list of members.
func (s *ProjectServices) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &s.Services)
}
