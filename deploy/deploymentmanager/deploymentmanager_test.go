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

package deploymentmanager

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ghodss/yaml"
)

func TestCreateOrUpdateDeployment(t *testing.T) {
	projID := "foo-project"

	deployment := &Deployment{
		Imports: []*Import{{Path: "path/to/foo-template"}},
		Resources: []*Resource{
			{
				Name: "foo-resource",
				Type: "path/to/foo-template",
				Properties: map[string]interface{}{
					"name":    "foo-resource",
					"foo-key": "foo-value",
				},
				Metadata: &Metadata{
					DependsOn: []string{"bar-resource"},
				},
			},
			{
				Name: "bar-resource",
				Type: "path/to/bar-template",
			},
		},
	}

	wantDeploymentYAML := `
imports:
- path: path/to/foo-template

resources:
- name: foo-resource
  type: path/to/foo-template
  properties:
    name: foo-resource
    foo-key: foo-value
  metadata:
    dependsOn:
    - bar-resource
- name: bar-resource
  type: path/to/bar-template
`

	tests := []struct {
		name                  string
		listDeploymentName    string
		wantDeploymentCommand []string
	}{
		{
			name:               "create",
			listDeploymentName: "some-random-deployment",
			wantDeploymentCommand: []string{
				"gcloud", "deployment-manager", "deployments", "create", "foo-deployment", "--project", projID,
			},
		}, {
			name:               "update",
			listDeploymentName: "foo-deployment",
			wantDeploymentCommand: []string{
				"gcloud", "deployment-manager", "deployments", "update", "foo-deployment",
				"--delete-policy", "ABANDON", "--project", projID,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			commander := &fakeCommander{
				listDeploymentName:    tc.listDeploymentName,
				wantDeploymentCommand: tc.wantDeploymentCommand,
			}

			cmdRun = commander.Run
			cmdCombinedOutput = commander.CombinedOutput

			if err := Upsert("foo-deployment", deployment, projID); err != nil {
				t.Fatalf("createOrUpdateDeployment = %v", err)
			}

			got := make(map[string]interface{})
			want := make(map[string]interface{})
			if err := yaml.Unmarshal(commander.gotConfigFileContents, &got); err != nil {
				t.Fatalf("yaml.Unmarshal got config: %v", err)
			}
			if err := yaml.Unmarshal([]byte(wantDeploymentYAML), &want); err != nil {
				t.Fatalf("yaml.Unmarshal want deployment config: %v", err)
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatalf("deployment yaml differs (-got +want):\n%v", diff)
			}
		})
	}
}

// TODO: simplify this
type fakeCommander struct {
	listDeploymentName    string
	wantDeploymentCommand []string

	gotConfigFileContents []byte
}

func (c *fakeCommander) Run(cmd *exec.Cmd) error {
	if !cmp.Equal(cmd.Args[:len(c.wantDeploymentCommand)], c.wantDeploymentCommand) {
		return fmt.Errorf("fake cmdRun: unexpected args: %v", cmd.Args)
	}
	configFile := cmd.Args[len(cmd.Args)-1] // config file is the last file
	var err error
	c.gotConfigFileContents, err = ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read %q: %v", configFile, err)
	}
	return nil
}

func (c *fakeCommander) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	listArgs := []string{"gcloud", "deployment-manager", "deployments", "list", "--format", "json"}
	if cmp.Equal(cmd.Args[:len(listArgs)], listArgs) {
		out := fmt.Sprintf(`[{"name": "%s"}]`, c.listDeploymentName)
		return []byte(out), nil
	}
	return nil, fmt.Errorf("fake cmdCombinedOutput: unexpected args: %v", cmd.Args)
}
