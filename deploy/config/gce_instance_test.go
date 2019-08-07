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
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestGCEInstance(t *testing.T) {
	instanceYAML := `
properties:
  name: foo-instance
  zone: us-east1-a
`

	ins := &config.GCEInstance{}
	if err := yaml.Unmarshal([]byte(instanceYAML), ins); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := ins.Init(); err != nil {
		t.Fatalf("ins.Init: %v", err)
	}

	got := make(map[string]interface{})
	want := make(map[string]interface{})
	byt, err := yaml.Marshal(ins)
	if err != nil {
		t.Fatalf("yaml.Marshal dataset: %v", err)
	}
	if err := yaml.Unmarshal(byt, &got); err != nil {
		t.Fatalf("yaml.Unmarshal got config: %v", err)
	}
	if err := yaml.Unmarshal([]byte(instanceYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
	}

	if gotName, wantName := ins.Name(), "foo-instance"; gotName != wantName {
		t.Errorf("ins.Name() = %v, want %v", gotName, wantName)
	}
}

func TestGCEInstance_CustomBootImage(t *testing.T) {
	instanceYAML := `
properties:
  name: foo-instance
  zone: us-east1-a
custom_boot_image:
  image_name: foo-OS
  gcs_path: foo-os/some-image.tar.gz
`

	ins := &config.GCEInstance{}
	if err := yaml.Unmarshal([]byte(instanceYAML), ins); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}

	if err := ins.Init(); err != nil {
		t.Fatalf("ins.Init: %v", err)
	}

	if got, want := ins.DiskImage, "global/images/foo-OS"; got != want {
		t.Errorf("disk image custom boot image: got %q, want %q", got, want)
	}
}
