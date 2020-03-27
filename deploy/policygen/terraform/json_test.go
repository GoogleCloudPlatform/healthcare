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
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const testPlanPath = "policygen/terraform/testdata/plan.json"
const testStatePath = "policygen/terraform/testdata/state.json"

// These functions will fatal out so there's no need to return errors.
func readTestFile(t *testing.T, path string) []byte {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan json file: %v", err)
	}
	return b
}

func unmarshalTestPlan(t *testing.T) *plan {
	b := readTestFile(t, testPlanPath)

	p := new(plan)
	if err := json.Unmarshal(b, p); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	return p
}

func TestReadPlanResources(t *testing.T) {
	b := readTestFile(t, testPlanPath)

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
	b := readTestFile(t, testStatePath)

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

func TestReadPlanChanges(t *testing.T) {
	b := readTestFile(t, testPlanPath)

	// Predefine the changes in the testdata file.
	addBucketChange := ResourceChange{
		Address: "google_storage_bucket.gcs_tf_bucket",
		Mode:    "managed",
		Kind:    "google_storage_bucket",
		Name:    "gcs_tf_bucket",
	}
	recreateAPIChange := ResourceChange{
		Address: "google_project_service.enable_container_registry",
		Mode:    "managed",
		Kind:    "google_project_service",
		Name:    "enable_container_registry",
	}

	tests := []struct {
		actions []string
		want    []ResourceChange
	}{
		// No actions - should get all changes.
		{[]string{}, []ResourceChange{addBucketChange, recreateAPIChange}},

		// Only one action - should get only that change.
		{[]string{"create"}, []ResourceChange{addBucketChange}},

		// Two actions - should get only the one with both actions in that order.
		{[]string{"delete", "create"}, []ResourceChange{recreateAPIChange}},

		// Two actions in a different order - should return nothing.
		{[]string{"create", "delete"}, []ResourceChange{}},
	}

	for _, testcase := range tests {
		changes, err := ReadPlanChanges(b, testcase.actions)
		if err != nil {
			t.Fatalf("unmarshal json: %v", err)
		}
		if diff := cmp.Diff(changes, testcase.want, cmpopts.IgnoreFields(ResourceChange{}, "Change"), cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ReadPlanChanges(%v, %v) returned diff (-got +want):\n%s", testPlanPath, testcase.actions, diff)
		}
	}
}

func TestResourceProviderConfig(t *testing.T) {
	p := unmarshalTestPlan(t)

	tests := []struct {
		kind    string
		name    string
		wantErr bool
	}{
		// Empty address - should return err.
		{"", "", true},

		// No provider for this resource - should return err.
		{"google_storage_bucket", "some_other_bucket", true},

		// Provider config exists - should not return err.
		{"google_storage_bucket", "gcs_tf_bucket", false},
	}
	for _, testcase := range tests {
		config, err := resourceProviderConfig(testcase.kind, testcase.name, p)
		if testcase.wantErr {
			if err == nil {
				t.Errorf("resourceProviderConfig(%q, %q, %v) returned nil err; wanted err != nil", testcase.kind, testcase.name, testPlanPath)
			}
		} else {
			if cmp.Equal(config, ProviderConfig{}, cmpopts.EquateEmpty()) {
				t.Errorf("resourceProviderConfig(%q, %q, %v) returned empty provider config; want nonempty config", testcase.kind, testcase.name, testPlanPath)
			}
		}
	}
}

func TestResolveReference(t *testing.T) {
	p := unmarshalTestPlan(t)

	tests := []struct {
		ref  string
		want interface{}
	}{
		// Empty reference - should return nil.
		{"", nil},

		// Nonexistent variable - should return nil.
		{"var.nonexistent", nil},

		// Existing variable - should return variable value.
		{"var.bucket_name", "test-bucket"},

		// No other reference types are supported at the moment.
	}
	for _, testcase := range tests {
		resolved := resolveReference(testcase.ref, p)
		if !cmp.Equal(resolved, testcase.want) {
			t.Errorf("resolveReference(%q, %v) = %v; want %v", testcase.ref, testPlanPath, resolved, testcase.want)
		}
	}
}

func TestResolveExpression(t *testing.T) {
	p := unmarshalTestPlan(t)

	emptyExpr := map[string]interface{}{}
	constantValue := "US"
	nonexistentRef := map[string]interface{}{
		"references": []interface{}{"nonexistent"},
	}

	tests := []struct {
		expr map[string]interface{}
		want interface{}
	}{
		// nil expression - should return nil.
		{nil, nil},

		// Empty expression - should return it as-is.
		{emptyExpr, emptyExpr},

		// constant_value - should return value directly.
		{
			map[string]interface{}{
				"constant_value": constantValue,
			},
			constantValue,
		},

		// references with nonexistent reference - should return as-is.
		{nonexistentRef, nonexistentRef},

		// references with one reference - should resolve.
		{
			map[string]interface{}{
				"references": []interface{}{
					"var.bucket_name",
				},
			},
			"test-bucket",
		},

		// references with multiple references - should resolve first one that matches.
		{
			map[string]interface{}{
				"references": []interface{}{
					"nonexistent",
					"anothernontexistent",
					"var.project",
					"var.bucket_name",
				},
			},
			"test-project",
		},
	}
	for _, testcase := range tests {
		resolved := resolveExpression(testcase.expr, p)
		if diff := cmp.Diff(resolved, testcase.want); diff != "" {
			t.Errorf("resolveExpression(%q, %v) returned diff (-got +want):\n%s", testcase.expr, testPlanPath, diff)
		}
	}
}

func TestReadProviderConfigValues(t *testing.T) {
	b := readTestFile(t, testPlanPath)

	kind := "google_storage_bucket"
	name := "gcs_tf_bucket"
	got, err := ReadProviderConfigValues(b, kind, name)
	if err != nil {
		t.Fatalf("ReadProviderConfigValues(%v, %q, %q) err: %v", testPlanPath, kind, name, err)
	}

	want := map[string]interface{}{
		"credentials": map[string]interface{}{},
		"location":    "US",
		"project":     "test-project",
		"region":      "us-central1",
		"zone":        "us-central1-c",
	}

	if diff := cmp.Diff(got, want, cmpopts.EquateEmpty()); diff != "" {
		t.Fatalf("retrieved provider configs differ (-got +want):\n%v", diff)
	}
}
