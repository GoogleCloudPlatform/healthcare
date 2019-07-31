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

package terraform

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestModule(t *testing.T) {
	got := make(map[string]interface{})
	want := map[string]interface{}{
		"source":     "foo",
		"prop-field": "bar",
	}
	b, err := json.Marshal(&Module{
		Source:     "foo",
		Properties: map[string]interface{}{"prop-field": "bar"},
	})
	if err != nil {
		t.Fatalf("json.Marshal properties: %v", err)
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal got config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}
}
