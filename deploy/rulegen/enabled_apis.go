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

import (
	"fmt"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

// EnabledAPIsRule represents a forseti enabled APIs rule.
type EnabledAPIsRule struct {
	Name      string     `yaml:"name"`
	Mode      string     `yaml:"mode"`
	Resources []resource `yaml:"resource"`
	Services  []string   `yaml:"services"`
}

// EnabledAPIsRules builds enabled APIs scanner rules for the given config.
func EnabledAPIsRules(conf *config.Config) ([]EnabledAPIsRule, error) {
	rules := []EnabledAPIsRule{{
		Name:      "Global API whitelist.",
		Mode:      "whitelist",
		Resources: []resource{{Type: "project", IDs: []string{"*"}}},
		Services:  conf.Overall.AllowedAPIs,
	}}

	for _, project := range conf.AllProjects() {
		if len(project.EnabledAPIs) == 0 {
			continue
		}
		rules = append(rules, EnabledAPIsRule{
			Name:      fmt.Sprintf("API whitelist for %s.", project.ID),
			Mode:      "whitelist",
			Resources: []resource{{Type: "project", IDs: []string{project.ID}}},
			Services:  project.EnabledAPIs,
		})
	}

	return rules, nil
}
