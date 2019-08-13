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

package apply

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
)

func TestForsetiConfig(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	var gotTFConf *terraform.Config
	terraformApply = func(config *terraform.Config, _ string, _ *terraform.Options) error {
		gotTFConf = config
		return nil
	}

	if err := ForsetiConfig(conf, true); err != nil {
		t.Fatalf("Forseti = %v", err)
	}

	wantConfig := `{
	"terraform": {
		"required_version": ">= 0.12.0",
		"backend": {
			"gcs": {
				"bucket": "my-forseti-project-state",
				"prefix": "forseti"
			}
		}
	},
	"module": [{
		"forseti": {
			"source": "./external/terraform_google_forseti",
			"composite_root_resources": [
			  "organizations/12345678",
				"folders/98765321"
			 ],
			 "domain": "my-domain.com",
			 "project_id": "my-forseti-project",
			 "storage_bucket_location": "us-east1"
		}
	}]
}`

	var got, want interface{}
	b, err := json.Marshal(gotTFConf)
	if err != nil {
		t.Fatalf("json.Marshal gotTFConf: %v", err)
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal got = %v", err)
	}
	if err := json.Unmarshal([]byte(wantConfig), &want); err != nil {
		t.Fatalf("json.Unmarshal want = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("terraform config differs (-got, +want):\n%v", diff)
	}
}

func TestGrantForsetiPermissions(t *testing.T) {
	wantCmdCnt := 9
	wantCmdPrefix := "gcloud projects add-iam-policy-binding project1 --member serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com --role roles/"
	var got []string
	cmdRun = func(cmd *exec.Cmd) error {
		got = append(got, strings.Join(cmd.Args, " "))
		return nil
	}
	if err := GrantForsetiPermissions("project1", "forseti-sa@@forseti-project.iam.gserviceaccount.com"); err != nil {
		t.Fatalf("GrantForsetiPermissions = %v", err)
	}
	if len(got) != wantCmdCnt {
		t.Fatalf("number of permissions granted differ: got %d, want %d", len(got), wantCmdCnt)
	}
	for _, cmd := range got {
		if !strings.HasPrefix(cmd, wantCmdPrefix) {
			t.Fatalf("command %q does not contain expected prefix %q", cmd, wantCmdPrefix)
		}
	}
}
