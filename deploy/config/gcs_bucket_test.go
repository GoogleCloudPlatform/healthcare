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

package config_test

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestGCSBucket(t *testing.T) {
	bucketYAML := `
ttl_days: 7
properties:
  name: foo-bucket
  location: us-east1
  bindings:
  - role: roles/storage.objectViewer
    members:
    - 'user:extra-reader@google.com'
  lifecycle:
    rule:
    - action:
        type: SetStorageClass
        storageClass: NEARLINE
      condition:
        age: 36500
        createdBefore: "2018-08-16"
        isLive: false
        matchesStorageClass:
        - REGIONAL
        - STANDARD
        - COLDLINE
        numNewerVersions: 5
`

	wantBucketYAML := `
ttl_days: 7
properties:
  name: foo-bucket
  location: us-east1
  bindings:
  - role: roles/storage.objectViewer
    members:
    - 'user:extra-reader@google.com'
  versioning:
    enabled: True
  lifecycle:
    rule:
    - action:
        type: SetStorageClass
        storageClass: NEARLINE
      condition:
        age: 36500
        createdBefore: "2018-08-16"
        isLive: false
        matchesStorageClass:
        - REGIONAL
        - STANDARD
        - COLDLINE
        numNewerVersions: 5
    - action:
        type: Delete
      condition:
        age: 7
        isLive: true
`

	b := &config.GCSBucket{}
	if err := yaml.Unmarshal([]byte(bucketYAML), b); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := b.Init(); err != nil {
		t.Fatalf("d.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	byt, err := yaml.Marshal(b)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(byt, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(wantBucketYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := b.Name(), "foo-bucket"; gotName != wantName {
		t.Errorf("d.Name() = %v, want %v", gotName, wantName)
	}
}

func TestGCSBucketErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		err  string
	}{
		{
			"missing_name",
			"properties	: {}",
			"name must be set",
		},
		{
			"missing_location",
			"properties: {name: foo-bucket}",
			"location must be set",
		},
		{
			"versioning_disabled",
			"properties: { name: foo-bucket, location: us-east1, versioning: { enabled: false }}",
			"versioning must not be disabled",
		},
		{
			"predefined_acl_set",
			"properties: { name: foo-bucket, location: us-east1, predefinedAcl: publicRead}",
			"predefined ACLs must not be set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &config.GCSBucket{}
			if err := yaml.Unmarshal([]byte(tc.yaml), b); err != nil {
				t.Fatalf("yaml unmarshal: %v", err)
			}
			if err := b.Init(); err == nil {
				t.Fatalf("b.Init error: got nil, want %v", tc.err)
			} else if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("b.Init: got error %q, want error with substring %q", err, tc.err)
			}
		})
	}
}
