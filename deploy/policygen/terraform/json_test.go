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

package terraform

import (
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestReadPlanResources(t *testing.T) {
	path := "policygen/terraform/testdata/plan.json"

	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan json file: %v", err)
	}

	resources, err := ReadPlanResources(b)
	if err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	if len(resources) != 12 {
		t.Fatalf("retrieved number of resources differ: got %d, want 12", len(resources))
	}

	// TODO: check more things.
}

func TestReadStateResources(t *testing.T) {
	path := "policygen/terraform/testdata/state.json"

	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("read state json file: %v", err)
	}

	resources, err := ReadStateResources(b)
	if err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	if len(resources) != 1 {
		t.Fatalf("retrieved number of resources differ: got %d, want 1", len(resources))
	}

	want := Resource{
		Name:    "allow-all-ingress",
		Address: "google_compute_firewall.allow-all-ingress",
		Kind:    "google_compute_firewall",
		Mode:    "managed",
	}

	if diff := cmp.Diff(resources[0], want, cmpopts.IgnoreFields(Resource{}, "Values")); diff != "" {
		t.Fatalf("retrieved resource differs (-got +want):\n%v", diff)
	}

}
