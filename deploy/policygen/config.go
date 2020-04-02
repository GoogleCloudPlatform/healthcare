/*
 * Copyright 2020 Google LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

// Config is the struct representing the Policy Generator configuration.
type Config struct {
	OrgID           string                 `json:"org_id"`
	ForsetiPolicies map[string]interface{} `json:"forseti_policies"`
	GCPOrgPolicies  map[string]interface{} `json:"gcp_organization_policies"`
}

func loadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %v", path, err)
	}
	c := new(Config)
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("unmarshal config %q: %v", path, err)
	}
	if c.OrgID == "" {
		return nil, fmt.Errorf("`org_id` is required")
	}
	return c, nil
}
