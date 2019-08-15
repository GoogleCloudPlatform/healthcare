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

package tfconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestStorageBucket(t *testing.T) {
	bucketYAML := `
name: foo-bucket
location: us-east1
`

	wantBucketYAML := `
name: foo-bucket
location: us-east1
project: foo-project
versioning:
  enabled: true
`

	b := &StorageBucket{}
	if err := yaml.Unmarshal([]byte(bucketYAML), b); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := b.Init("foo-project"); err != nil {
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

	if gotID, wantID := b.ID(), "foo-bucket"; gotID != wantID {
		t.Errorf("d.Name() = %v, want %v", gotID, wantID)
	}
}
