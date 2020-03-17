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

package tfplan

import (
	"io/ioutil"
	"testing"
)

func TestReadJSONResources(t *testing.T) {
	planPath := "policygen/tfplan/testdata/test.plan.json"

	b, err := ioutil.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read plan json file: %v", err)
	}

	resources, err := ReadJSONResources(b)
	if err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	if len(resources) != 12 {
		t.Fatalf("retrieved number of resources differ: got %d, want 12", len(resources))
	}

	// TODO: check more things.
}
