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

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/ghodss/yaml"
)

func TestDefaultResource(t *testing.T) {
	resYAML := `
properties:
  name: foo-resource
`

	d := new(config.DefaultResource)
	if err := yaml.Unmarshal([]byte(resYAML), d); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	d.TmplPath = "foo"
	if err := d.Init(); err != nil {
		t.Fatalf("d.Init = %v", err)
	}
	if d.Name() != "foo-resource" {
		t.Fatalf("default resource name error: %v", d.Name())
	}
}
