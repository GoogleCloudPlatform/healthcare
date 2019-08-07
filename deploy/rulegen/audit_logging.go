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

// AuditLoggingRule represents a forseti audit logging rule.
type AuditLoggingRule struct {
	Name      string     `yaml:"name"`
	Resources []resource `yaml:"resource"`
	Service   string     `yaml:"service"`
	LogTypes  []string   `yaml:"log_types"`
}

// AuditLoggingRules builds audit logging scanner rules for the given config.
func AuditLoggingRules(conf *config.Config) ([]AuditLoggingRule, error) {
	return []AuditLoggingRule{{
		Name: "Require all Cloud Audit logs.",
		Resources: []resource{{
			Type: "project",
			IDs:  []string{"*"},
		}},
		Service: "allServices",
		LogTypes: []string{
			"ADMIN_READ",
			"DATA_READ",
			"DATA_WRITE",
		},
	}}, nil
}
