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
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

type migrationsRunner struct {
	state []byte

	apiRemoved bool
}

// CmdRun prints the command information.
func (r *migrationsRunner) CmdRun(cmd *exec.Cmd) error {
	cmdStr := strings.Join(cmd.Args, " ")
	if cmdStr != `terraform state rm google_project_service.services["bigquery-json.googleapis.com"]` {
		return fmt.Errorf("unexpected call: %v", cmdStr)
	}
	r.apiRemoved = true
	return nil
}

// CmdOutput prints the command information.
func (r *migrationsRunner) CmdOutput(cmd *exec.Cmd) ([]byte, error) {
	cmdStr := strings.Join(cmd.Args, " ")
	if !contains(cmdStr, "terraform state list") {
		return nil, fmt.Errorf("unexpected call: %v", cmdStr)
	}
	return r.state, nil
}

func (*migrationsRunner) CmdCombinedOutput(*exec.Cmd) ([]byte, error) {
	return nil, nil
}

func TestRemoveDeprecatedBigqueryAPI(t *testing.T) {
	cases := []struct {
		name        string
		state       string
		wantRemoved bool
	}{
		{
			name:        "no_api",
			wantRemoved: false,
		},
		{
			name:        "has_api",
			state:       `google_project_service.services["bigquery-json.googleapis.com"]`,
			wantRemoved: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rn := &migrationsRunner{state: []byte(tc.state)}

			if err := RemoveDeprecatedBigqueryAPI("", rn); err != nil {
				t.Fatalf("RemoveDeprecatedBigqueryAPI: %v", err)
			}

			if rn.apiRemoved != tc.wantRemoved {
				t.Errorf("bigquery api removed: got %v, want %v", rn.apiRemoved, tc.wantRemoved)
			}
		})
	}
}
