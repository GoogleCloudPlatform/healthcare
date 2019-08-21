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
// See https://www.terraform.io/docs/configuration/syntax-json.html for documentation.
type Config struct {
	Terraform Terraform   `json:"terraform"`
	Modules   []*Module   `json:"module,omitempty"`
	Resources []*Resource `json:"resource,omitempty"`
}

// NewConfig returns a new terraform config.
func NewConfig() *Config {
	c := &Config{
		Terraform: Terraform{
			RequiredVersion: ">= 0.12.0",
		},
	}
	return c
}

// Terraform provides a terraform block config.
// See https://www.terraform.io/docs/configuration/terraform.html for details.
type Terraform struct {
	RequiredVersion string   `json:"required_version,omitempty"`
	Backend         *Backend `json:"backend,omitempty"`
}

// Backend provides a terraform backend config.
// See https://www.terraform.io/docs/backends/types/gcs.html.
type Backend struct {
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix,omitempty"`
}

// MarshalJSON implements a custom marshaller which marshals the backend under a "gcs" block.
func (b *Backend) MarshalJSON() ([]byte, error) {
	type alias Backend // use type alias to avoid infinite recursion
	return json.Marshal(map[string]interface{}{"gcs": alias(*b)})
}

// Module provides a terraform module config.
// See https://www.terraform.io/docs/configuration/modules.html for details.
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

// Resource defines a terraform resource config.
//
// Dependencies can be created between resources using terraform references.
// See https://www.terraform.io/docs/configuration/resources.html#resource-dependencies.
//
// Meta-arguments are also supported.
// See https://www.terraform.io/docs/configuration/resources.html#meta-arguments
// This is especially useful when wanting to create multiple resources that share a common set of fields,
// such as IAM members of a data resource (e.g. storage_bucket_iam_member).
// Instead of creating a separate config for each IAM member, a single IAM member using a meta-argument
// like for_each or count can be expanded by terraform to deploy all IAM members of a resource.
type Resource struct {
	Name       string
	Type       string
	Properties interface{}
}

// MarshalJSON implements a custom marshaller which marshals properties to the top level.
func (r *Resource) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		r.Type: map[string]interface{}{
			r.Name: r.Properties,
		},
	})
}

// Import defines fields used for a terraform import.
// See https://www.terraform.io/docs/import/usage.html.
type Import struct {
	Address string
	ID      string
}
