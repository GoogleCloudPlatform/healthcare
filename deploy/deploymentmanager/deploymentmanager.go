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

// Package deploymentmanager provides utilities for deployment manager.
package deploymentmanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/ghodss/yaml"
)

// Deployment represents a single deployment which can be used by the GCP Deployment Manager.
// TODO: move into separate package.
type Deployment struct {
	Imports   []*Import   `json:"imports,omitempty"`
	Resources []*Resource `json:"resources"`
}

// Import respresents a deployment manager template import.
type Import struct {
	Path string `json:"path"`
}

// Resource defines the deployment manager resources to deploy.
type Resource struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Metadata   *Metadata              `json:"metadata,omitempty"`
}

// Metadata contains extra metadata of the deployment.
type Metadata struct {
	DependsOn []string `json:"dependsOn,omitempty"`
}

// Upsert creates the deployment if it does not exist, else updates it.
func Upsert(name string, deployment *Deployment, projectID string, runner runner.Runner) error {
	b, err := yaml.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment : %v", err)
	}
	log.Printf("Creating deployment %q:\n%v", name, string(b))

	tmp, err := ioutil.TempFile("", "")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(b); err != nil {
		return fmt.Errorf("failed to write deployment to file: %v", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}

	exists, err := checkDeploymentExists(name, projectID, runner)
	if err != nil {
		return fmt.Errorf("failed to check if deployment exists: %v", err)
	}

	args := []string{"deployment-manager", "deployments"}
	if exists {
		// Due to the sensitive nature of the resources we manage, we don't want to
		// delete any resources after they have been deployed. Instead, abandon the resource
		// so the user can manually delete them later on.
		args = append(args, "update", name, "--delete-policy", "ABANDON")
	} else {
		args = append(args, "create", name)
	}
	args = append(args, "--project", projectID, "--config", tmp.Name())

	log.Printf("Running gcloud command with args: %v", args)

	cmd := exec.Command("gcloud", args...)
	if err := runner.CmdRun(cmd); err != nil {
		return fmt.Errorf("failed to run command: %v", err)
	}
	return nil
}

// checkDeploymentExists determines whether the deployment with the given name exists in the given project.
func checkDeploymentExists(name, projectID string, runner runner.Runner) (bool, error) {
	type deploymentInfo struct {
		Name string `json:"name"`
	}

	cmd := exec.Command("gcloud", "deployment-manager", "deployments", "list", "--format", "json", "--project", projectID)

	out, err := runner.CmdCombinedOutput(cmd)
	if err != nil {
		return false, fmt.Errorf("failed to run command: %v\n%v", err, string(out))
	}

	deploymentInfos := make([]deploymentInfo, 0)
	if err := json.Unmarshal(out, &deploymentInfos); err != nil {
		return false, fmt.Errorf("failed to unmarshal deployment list call: %v", err)
	}

	log.Printf("found %v deployments: %v", len(deploymentInfos), deploymentInfos)

	for _, d := range deploymentInfos {
		if d.Name == name {
			return true, nil
		}
	}
	return false, nil
}
