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

package resources

import (
	"testing"
)

func TestFromConfigValues(t *testing.T) {
	configs := []ProviderConfigMap{
		ProviderConfigMap{
			"":       "emptyValFirst",
			"mykey1": "key1First",
			"mykey2": "key2First",
		},
		nil,                 // nil map to ensure it doesn't crash the check.
		ProviderConfigMap{}, // Empty map just to make sure it gets skipped correctly.
		ProviderConfigMap{
			"mykey1": "key1Second",
			"mykey3": "key3Second",
		},
	}

	tests := []struct {
		key  string
		cvs  []ProviderConfigMap
		want interface{}
	}{
		// Empty configs - should return nil.
		{"", nil, nil},

		// Empty key - should still work.
		{"", configs, "emptyValFirst"},

		// Key not in any config - should return nil.
		{"nonExistentKey", configs, nil},

		// Key in second config - should return second one, not nil.
		{"mykey3", configs, "key3Second"},

		// Key in both configs - should return first one.
		{"mykey1", configs, "key1First"},
	}
	for _, tc := range tests {
		got := fromConfigValues(tc.key, tc.cvs...)
		if got != tc.want {
			t.Errorf("fromConfigValues(%v, %v) = %v; want %v", tc.key, tc.cvs, got, tc.want)
		}
	}
}
