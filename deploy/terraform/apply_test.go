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

	"github.com/google/go-cmp/cmp"
)

func TestApply(t *testing.T) {
	var b []byte
	cmdRun = func(cmd *exec.Cmd) error {
		if cmd.Args[0] == "terraform" && len(b) == 0 {
			var err error
			b, err = ioutil.ReadFile(filepath.Join(cmd.Dir, "main.tf.json"))
			if err != nil {
				t.Fatalf("ioutil.ReadFile = %v", err)
			}
		}
		return nil
	}

	conf := &Config{
		Modules: []*Module{{
			Name:       "foo-module",
			Source:     "foo-source",
			Properties: map[string]interface{}{"foo-prop": "foo-val"},
		}},
	}
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("ioutil.TempDir: %v", err)
	}
	defer os.RemoveAll(dir)
	if err := Apply(conf, dir, nil); err != nil {
		t.Fatalf("Forseti = %v", err)
	}

	wantConfig := `{
	"module": [{
		"foo-module": {
			 "source": "foo-source",
			 "foo-prop": "foo-val"
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
