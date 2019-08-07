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

// LienRule represents a forseti lien rule.
type LienRule struct {
	Name         string     `yaml:"name"`
	Mode         string     `yaml:"mode"`
	Resources    []resource `yaml:"resource"`
	Restrictions []string   `yaml:"restrictions"`
}

// LienRules builds lien scanner rules for the given config.
func LienRules(conf *config.Config) ([]LienRule, error) {
	return []LienRule{{
		Name:         "Require project deletion liens for all projects.",
		Mode:         "required",
		Resources:    []resource{globalResource(conf)},
		Restrictions: []string{"resourcemanager.projects.delete"},
	}}, nil
}
