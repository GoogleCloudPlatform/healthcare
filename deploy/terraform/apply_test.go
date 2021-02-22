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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type testRunner struct {
	gotConfig []byte
}

func (r *testRunner) CmdRun(cmd *exec.Cmd) error {
	if cmd.Args[0] == "terraform" && len(r.gotConfig) == 0 {
		var err error
		r.gotConfig, err = ioutil.ReadFile(filepath.Join(cmd.Dir, "main.tf.json"))
		if err != nil {
			return fmt.Errorf("ioutil.ReadFile = %v", err)
		}
	}
	return nil
}

func (*testRunner) CmdOutput(*exec.Cmd) ([]byte, error) { return nil, nil }

func (*testRunner) CmdCombinedOutput(*exec.Cmd) ([]byte, error) { return nil, nil }

func TestApply(t *testing.T) {
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

	opts := &Options{}

	customJSON := []byte(`{
  "resource": [{
     "custom_type": {
		   "custom-name": {
         "name": "custom-name",
         "project": "my-project"
			 }
     }
  }]
}`)
	if err := json.Unmarshal(customJSON, &opts.CustomConfig); err != nil {
		t.Fatalf("json.Unmarshal custom JSON: %v", err)
	}
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("ioutil.TempDir: %v", err)
	}
	defer os.RemoveAll(dir)
	r := &testRunner{}
	if err := Apply(conf, dir, opts, r); err != nil {
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
	"resource": [
    {
		  "foo-type": {
		  	"foo-resource": {
		  		"prop1": "val1"
	  		}
	  	}
	  },
    {
      "custom_type": {
        "custom-name": {
          "name": "custom-name",
          "project": "my-project"
        }
      }
    }
  ],
	"module": [{
		"foo-module": {
			 "source": "foo-source",
			 "prop2": "val2"
		}
	}]
}`

	var got, want interface{}
	if err := json.Unmarshal(r.gotConfig, &got); err != nil {
		t.Fatalf("json.Unmarshal = %v", err)
	}
	if err := json.Unmarshal([]byte(wantConfig), &want); err != nil {
		t.Fatalf("json.Unmarshal = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("terraform config differs (-got, +want):\n%v", diff)
	}
}
