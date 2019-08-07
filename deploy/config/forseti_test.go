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
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestForseti(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	var got interface{}
	want := map[string]interface{}{
		"project_id": "my-forseti-project",
		"domain":     "my-domain.com",
		"composite_root_resources": []interface{}{
			"organizations/12345678",
			"folders/98765321",
		},
		"storage_bucket_location": "us-east1",
	}
	b, err := yaml.Marshal(conf.Forseti.Properties)
	if err != nil {
		t.Fatalf("yaml.Marshal properties: %v", err)
	}
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}
}
