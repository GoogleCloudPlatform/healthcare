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

// Package resources defines resource-specific implementations for interface Importer.
package resources

// ProviderConfigMap is a type alias to make variables more readable.
type ProviderConfigMap map[string]interface{}

// fromConfigValues returns the first matching config value for key, from the given config value maps cvs.
func fromConfigValues(key string, cvs ...ProviderConfigMap) interface{} {
	for _, cv := range cvs {
		if v, ok := cv[key]; ok {
			return v
		}
	}
	return nil
}
