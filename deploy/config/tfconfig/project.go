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
func (p *ProjectResource) ID() string {
	return p.ProjectID
}

// ResourceType returns the resource terraform provider type.
func (p *ProjectResource) ResourceType() string {
	return "google_project"
}

// ImportID returns the ID to use for terraform imports.
func (p *ProjectResource) ImportID() string {
	return p.ProjectID
}
