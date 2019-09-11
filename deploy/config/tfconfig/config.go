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

// Package tfconfig provides utilities to parse terraform resource configurations.
// This is a temporary package and will be merged back into the parent config package once deployment manager has been deprecated.
package tfconfig

import (
	"regexp"
	"strings"
)

// Resource is an interface that must be implemented by all concrete resource implementations.
type Resource interface {
	Init(projectID string) error
	ID() string
	ResourceType() string
}

// invalidIDRE defines the invalid characters not allowed in terraform resource names.
var invalidIDRE = regexp.MustCompile("[^a-z0-9-_]")

// standardizeID replaces all characters not allowed for terraform resource names with underscores.
// It will also lowercase the name.
func standardizeID(id string) string {
	return invalidIDRE.ReplaceAllString(strings.ToLower(id), "_")
}
