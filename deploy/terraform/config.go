/*
 * Copyright 2019 Google LLC.
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

// Package terraform provides helpers for working with Terraform.
package terraform

import (
	"encoding/json"
	"fmt"
)

// Config represents a Terraform config.
// See https://www.terraform.io/docs/configuration/syntax-json.htm for documentation.
type Config struct {
	Modules []*Module `json:"module"`
}

// Module provides a terraform module config.
type Module struct {
	Name       string      `json:"-"`
	Source     string      `json:"source"`
	Properties interface{} `json:"-"`
}

// MarshalJSON implements a custom marshaller which marshals properties to the top level.
func (m *Module) MarshalJSON() ([]byte, error) {
	type alias Module // use type alias to avoid infinite recursion
	b, err := json.Marshal(alias(*m))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal module: %v", err)
	}

	merged := make(map[string]interface{})
	if err := json.Unmarshal(b, &merged); err != nil {
		return nil, fmt.Errorf("failed to unmarshal module bytes: %v", err)
	}

	b, err = json.Marshal(m.Properties)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal properties: %v", err)
	}
	if err := json.Unmarshal(b, &merged); err != nil {
		return nil, fmt.Errorf("failed to unmarshal properties: %v", err)
	}
	return json.Marshal(map[string]interface{}{
		m.Name: merged,
	})
}
