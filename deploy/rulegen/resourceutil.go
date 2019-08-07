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

package rulegen

import "github.com/GoogleCloudPlatform/healthcare/deploy/config"

type resource struct {
	Type      string   `yaml:"type,omitempty"`
	AppliesTo string   `yaml:"applies_to,omitempty"`
	IDs       []string `yaml:"resource_ids"`
}

// globalResource tries to find the broadest scope of resources defined in the config.
// This is required due to organization and folder IDs being optional.
// The order of preference is organization, folders or all projects defined in the config.
func globalResource(conf *config.Config) resource {
	switch {
	case conf.Overall.OrganizationID != "":
		return resource{Type: "organization", IDs: []string{conf.Overall.OrganizationID}}
	case conf.Overall.FolderID != "":
		return resource{Type: "folder", IDs: conf.AllFolders()}
	default:
		var ids []string
		for _, p := range conf.AllProjects() {
			ids = append(ids, p.ID)
		}
		return resource{Type: "project", IDs: ids}
	}
}
