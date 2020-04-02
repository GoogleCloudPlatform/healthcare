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
	"resourceName": resourceName,
}

func get(d map[string]interface{}, key string) interface{} {
	if val, ok := d[key]; ok {
		return val
	}
	return nil
}

func resourceName(s string) string {
	return strings.Replace(s, "-", "_", -1)
}
