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

package rulegen

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

type testRunner struct {
	args []string
}

func (*testRunner) CmdRun(*exec.Cmd) error { return nil }

func (*testRunner) CmdOutput(*exec.Cmd) ([]byte, error) { return nil, nil }

func (r *testRunner) CmdCombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	if len(r.args) != 0 {
		return nil, errors.New("fake CombinedOutput: unexpectedly called more than once")
	}
	r.args = cmd.Args
	if len(r.args) != 4 {
		return nil, fmt.Errorf("fake CombinedOutput: unexpected number of args: got %d, want 4, %v ", len(cmd.Args), cmd.Args)
	}
	checkRulesDir(&testing.T{}, filepath.Dir(r.args[2]))
	return nil, nil
}

func TestRunOutputPath(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("ioutil.TempDir = %v", err)
	}

	if err := Run(conf, tmpDir, DefaultAuditConfigFile, &testRunner{}); err != nil {
		t.Fatalf("Run = %v", err)
	}

	checkRulesDir(t, tmpDir)
}

func TestRunServerBucket(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)
	r := &testRunner{}
	if err := Run(conf, "", DefaultAuditConfigFile, r); err != nil {
		t.Fatalf("Run = %v", err)
	}
	wantRE, err := regexp.Compile(`gsutil cp .*\*\.yaml gs://my-forseti-project-server/rules`)
	if err != nil {
		t.Fatalf("regexp.Compile = %v", err)
	}
	if got := strings.Join(r.args, " "); !wantRE.MatchString(got) {
		t.Fatalf("rules upload command does not match: got %q, want match of %q", got, wantRE)
	}
}

func checkRulesDir(t *testing.T, rulesDir string) {
	t.Helper()

	// check one rules file
	b, err := ioutil.ReadFile(filepath.Join(rulesDir, "audit_logging_rules.yaml"))
	if err != nil {
		t.Fatalf("ioutil.ReadFile = %v", err)
	}

	wantYAML := `
rules:
- name: Require all Cloud Audit logs.
  resource:
  - type: project
    resource_ids:
    - '*'
  service: allServices
  log_types:
  - ADMIN_READ
  - DATA_READ
  - DATA_WRITE
`
	got := make(map[string]interface{})
	want := make(map[string]interface{})

	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Fatalf("audit logging rules differ (-got, +want):\n%v", diff)
	}
}
