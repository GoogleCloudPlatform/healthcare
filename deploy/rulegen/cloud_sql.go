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
	"strconv"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
)

// CloudSQLRule represents a forseti cloud SQL rule.
type CloudSQLRule struct {
	Name               string     `yaml:"name"`
	Resources          []resource `yaml:"resource"`
	InstanceName       string     `yaml:"instance_name"`
	AuthorizedNetworks string     `yaml:"authorized_networks"`
	SSLEnabled         string     `yaml:"ssl_enabled"`
}

// CloudSQLRules builds cloud SQL scanner rules for the given config.
func CloudSQLRules(conf *config.Config) ([]CloudSQLRule, error) {
	var rules []CloudSQLRule
	for _, b := range []bool{false, true} {
		ssl := "enabled"
		if !b {
			ssl = "disabled"
		}

		rules = append(rules, CloudSQLRule{
			Name:               fmt.Sprintf("Disallow publicly exposed cloudsql instances (SSL %s).", ssl),
			Resources:          []resource{globalResource(conf)},
			InstanceName:       "*",
			AuthorizedNetworks: "0.0.0.0/0",
			SSLEnabled:         strconv.FormatBool(b),
		})
	}
	return rules, nil
}
