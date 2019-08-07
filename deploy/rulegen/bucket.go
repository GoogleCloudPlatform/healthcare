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

// BucketRule represents a forseti GCS bucket ACL rule.
type BucketRule struct {
	Name      string     `yaml:"name"`
	Bucket    string     `yaml:"bucket"`
	Entity    string     `yaml:"entity"`
	Email     string     `yaml:"email"`
	Domain    string     `yaml:"domain"`
	Role      string     `yaml:"role"`
	Resources []resource `yaml:"resource"`
}

// BucketRules builds bucket scanner rules for the given config.
func BucketRules(conf *config.Config) ([]BucketRule, error) {
	return []BucketRule{{
		Name:      "Disallow all acl rules, only allow IAM.",
		Bucket:    "*",
		Entity:    "*",
		Email:     "*",
		Domain:    "*",
		Role:      "*",
		Resources: []resource{{IDs: []string{"*"}}},
	}}, nil
}
