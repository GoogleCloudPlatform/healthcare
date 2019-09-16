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
	"fmt"
)

const defaultRestriction = "resourcemanager.projects.delete"

// ResourceManagerLien supports Terraform liens.
// TODO: support imports for this resource.
type ResourceManagerLien struct {
	Origin       string   `json:"origin"`
	Parent       string   `json:"parent"`
	Restrictions []string `json:"restrictions"`
	Reason       string   `json:"reason"`

	ProjectDeletion bool `json:"_project_deletion"`
}

// Init initializes the resource.
func (l *ResourceManagerLien) Init(projectID string) error {
	if !l.ProjectDeletion {
		return nil
	}

	if l.Origin != "" {
		return fmt.Errorf("origin must be unset if _project_deletion is set: %v", l.Origin)
	}
	l.Origin = "managed-terraform"

	if l.Parent != "" {
		return fmt.Errorf("parent must be unset if _project_deletion is set: %v", l.Parent)
	}
	l.Parent = "projects/" + projectID

	if len(l.Restrictions) > 0 {
		return fmt.Errorf("restructions must be unset if _project_deletion is set: %v", l.Restrictions)
	}
	l.Restrictions = []string{defaultRestriction}

	if l.Reason != "" {
		return fmt.Errorf("reason must be unset if _project_deletion is set: %v", l.Reason)
	}
	l.Reason = "Managed project deletion lien"
	return nil
}

// ID returns the resource unique identifier.
func (l *ResourceManagerLien) ID() string {
	return standardizeID(l.Reason)
}

// ResourceType returns the resource terraform provider type.
func (*ResourceManagerLien) ResourceType() string {
	return "google_resource_manager_lien"
}

// MarshalJSON provides a custom JSON marshaller.
// It is used to remove private fields from the struct.
func (l *ResourceManagerLien) MarshalJSON() ([]byte, error) {
	type aliasResourceManagerLien ResourceManagerLien
	return interfacePair{nil, aliasResourceManagerLien(*l)}.MarshalJSON()
}
