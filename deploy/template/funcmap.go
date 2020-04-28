// Copyright 2020 Google LLC.
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

package template

import (
	"strings"
)

var funcMap = map[string]interface{}{
	"get":          get,
	"has":          has,
	"enabled":      enabled,
	"resourceName": resourceName,
}

// get allows a template to optionally lookup a value from a dict.
// If a value is not found, return nil.
//   {{if get . "OPTIONAL_KEY"}} {{.}} {{end}}  // GOOD
//   {{.OPTIONAL_KEY}}  // BAD, will fail if the key is not set
//
// Keys can reference multiple levels of maps by using "." to indicate a new level
// (e.g. L1.L2 will lookup key L1 in the top level map then L2 within the value.)
func get(m map[string]interface{}, key string) interface{} {
	split := strings.Split(key, ".")
	for i, k := range split {
		v, ok := m[k]
		switch {
		case !ok:
			return nil
		case i == len(split)-1:
			return v
		default:
			m = v.(map[string]interface{})
		}
	}
	return nil
}

// has determines whether the key is found in the given map.
// Keys can reference multiple levels of maps by using "." to indicate a new level
// (e.g. L1.L2 will lookup key L1 in the top level map then L2 within the value.)
func has(m map[string]interface{}, key string) bool {
	return get(m, key) != nil
}

// enabled is a helper to cleanly check if a key is set and its value is set to false.
// This is useful for checks for DISABLED keys.
func enabled(m map[string]interface{}, key string) bool {
	v := get(m, "DISABLED."+key)
	return v == nil || !v.(bool)
}

// resourceName builds a Terraform resource name.
// GCP resource names often use "-" but Terraform resource names should use "_".
func resourceName(s string) string {
	return strings.Replace(s, "-", "_", -1)
}
