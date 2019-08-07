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

package config

// Binding represents a GCP policy binding.
type Binding struct {
	Role    string   `json:"role" yaml:"role"`
	Members []string `json:"members" yaml:"members"`
}

// MergeBindings merges bindings together. It is typically used to merge default bindings with user specified bindings.
// Roles will be de-duplicated and merged into a single binding. Members are de-duplicated by deployment manager.
func MergeBindings(bs ...Binding) []Binding {
	roleToMembers := make(map[string][]string)
	var roles []string // preserve ordering

	for _, b := range bs {
		if _, ok := roleToMembers[b.Role]; !ok {
			roles = append(roles, b.Role)
		}
		roleToMembers[b.Role] = append(roleToMembers[b.Role], b.Members...)
	}

	var merged []Binding
	for _, role := range roles {
		merged = append(merged, Binding{Role: role, Members: roleToMembers[role]})
	}
	return merged
}
