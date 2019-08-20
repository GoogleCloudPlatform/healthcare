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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/google/go-cmp/cmp"
)

func TestApply(t *testing.T) {
	var b []byte
	origCmdRun := runner.CmdRun
	defer func() { runner.CmdRun = origCmdRun }()
	runner.CmdRun = func(cmd *exec.Cmd) error {
		if cmd.Args[0] == "terraform" && len(b) == 0 {
			var err error
			b, err = ioutil.ReadFile(filepath.Join(cmd.Dir, "main.tf.json"))
			if err != nil {
				t.Fatalf("ioutil.ReadFile = %v", err)
			}
		}
		return nil
	}

	conf := NewConfig()
	conf.Terraform.Backend = &Backend{
		Bucket: "foo-state",
		Prefix: "foo-prefix",
	}
	conf.Resources = []*Resource{{
		Name:       "foo-resource",
		Type:       "foo-type",
		Properties: map[string]interface{}{"prop1": "val1"},
	}}
	conf.Modules = []*Module{{
		Name:       "foo-module",
		Source:     "foo-source",
		Properties: map[string]interface{}{"prop2": "val2"},
	}}
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("ioutil.TempDir: %v", err)
	}
	defer os.RemoveAll(dir)
	if err := Apply(conf, dir, nil); err != nil {
		t.Fatalf("Forseti = %v", err)
	}

	wantConfig := `{
	"terraform": {
		"required_version": ">= 0.12.0",
		"backend": {
			"gcs": {
				"bucket": "foo-state",
				"prefix": "foo-prefix"
			}
		}
	},
	"resource": [{
		"foo-type": {
			"foo-resource": {
				"prop1": "val1"
			}
		}
	}],
	"module": [{
		"foo-module": {
			 "source": "foo-source",
			 "prop2": "val2"
		}
	}]
}`

	var got, want interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal = %v", err)
	}
	if err := json.Unmarshal([]byte(wantConfig), &want); err != nil {
		t.Fatalf("json.Unmarshal = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("terraform config differs (-got, +want):\n%v", diff)
	}

	// TODO: test with actual modules
}
