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

// Package apply provides utilities to apply a project config to GCP by deploying all defined resources.
package apply

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
)

const (
	deploymentNamePrefix            = "data-protect-toolkit"
	auditDeploymentName             = deploymentNamePrefix + "-audit"
	resourceDeploymentName          = deploymentNamePrefix + "-resources"
	setupPrerequisiteDeploymentName = deploymentNamePrefix + "-prerequisites"
)

// deploymentManagerRoles are the roles granted to the DM service account.
var deploymentManagerRoles = []string{"owner", "storage.admin"}

// deploymentRetryWaitTime is the time to wait between retrying a deployment to allow for concurrent operations to finish.
const deploymentRetryWaitTime = time.Minute

// The following vars are stubbed in tests.
var (
	terraformApply = terraform.Apply
)

// depender is the interface that defines a method to get dependent resources.
type depender interface {
	// Dependencies returns the name of the resource IDs to depend on.
	Dependencies() []string
}

func collectGCEInfo(project *config.Project, rn runner.Runner) error {
	cmd := exec.Command("gcloud", "--project", project.ID, "compute", "instances", "list", "--format", "json")
	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to list existing compute instances: %v", err)
	}

	var i []config.GCEInstanceInfo
	if err := json.Unmarshal(out, &i); err != nil {
		return fmt.Errorf("failed to unmarshal existing compute instances list output: %v", err)
	}
	project.GeneratedFields.GCEInstanceInfoList = i
	return nil
}

func getLogSinkServiceAccount(project *config.Project, sinkName string, rn runner.Runner) (string, error) {
	cmd := exec.Command("gcloud", "logging", "sinks", "describe", sinkName, "--format", "json", "--project", project.ID)

	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to query log sink service account from gcloud: %v", err)
	}

	type sink struct {
		WriterIdentity string `json:"writerIdentity"`
	}

	s := new(sink)
	if err := json.Unmarshal(out, s); err != nil {
		return "", fmt.Errorf("failed to unmarshal sink output: %v", err)
	}
	return strings.TrimPrefix(s.WriterIdentity, "serviceAccount:"), nil
}

func removeOwnerUser(project *config.Project, rn runner.Runner) error {
	cmd := exec.Command("gcloud", "config", "get-value", "account", "--format", "json", "--project", project.ID)
	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to get currently authenticated user: %v", err)
	}
	var member string
	if err := json.Unmarshal(out, &member); err != nil {
		return fmt.Errorf("failed to unmarshal current user: %v", err)
	}
	role := "roles/owner"
	member = "user:" + member

	if project.IAMMembers != nil {
		for _, m := range project.IAMMembers.Members {
			if m.Role == role && m.Member == member {
				// User owner specifically requested, so don't remove them.
				return nil
			}
		}
	}

	has, err := hasBinding(project, role, member, rn)
	if err != nil {
		return err
	}
	if !has {
		log.Printf("owner user %q already removed", member)
		return nil
	}

	cmd = exec.Command(
		"gcloud", "projects", "remove-iam-policy-binding", project.ID,
		"--member", member, "--role", role, "--project", project.ID)
	return rn.CmdRun(cmd)
}

func hasBinding(project *config.Project, role string, member string, rn runner.Runner) (has bool, err error) {
	cmd := exec.Command(
		"gcloud", "projects", "get-iam-policy", project.ID,
		"--project", project.ID,
		"--format", "json",
	)
	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return false, fmt.Errorf("failed to get iam policy bindings: %v", err)
	}
	log.Printf("Looking for role %q, member %q in:\n%v", role, member, string(out))

	type policy struct {
		Bindings []config.Binding `json:"bindings"`
	}
	p := new(policy)
	if err := json.Unmarshal(out, p); err != nil {
		return false, fmt.Errorf("failed to unmarshal get-iam-policy output: %v", err)
	}
	for _, b := range p.Bindings {
		if b.Role == role {
			for _, m := range b.Members {
				if m == member {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// TODO use Terraform once https://github.com/terraform-providers/terraform-provider-google/issues/2605 is resolved.
// createStackdriverAccount prompts the user to create a new Stackdriver Account.
func createStackdriverAccount(project *config.Project, rn runner.Runner) error {
	if project.StackdriverAlertEmail == "" && len(project.NotificationChannels) == 0 {
		log.Println("No Stackdriver alert email or notification channels specified, skipping creation of Stackdriver account.")
		return nil
	}
	exist, err := stackdriverAccountExists(project.ID, rn)
	if err != nil {
		return err
	}
	if exist {
		log.Println("Stackdriver account already exists.")
		return nil
	}

	message := fmt.Sprintf(`
------------------------------------------------------------------------------
To create email alerts, this project needs a Stackdriver account.
Create a new Stackdriver account for this project by visiting:
    https://console.cloud.google.com/monitoring?project=%s

Only add this project, and skip steps for adding additional GCP or AWS
projects. You don't need to install Stackdriver Agents.

IMPORTANT: Wait about 5 minutes for the account to be created.

For more information, see: https://cloud.google.com/monitoring/accounts/

After the account is created, type "yes" to continue, or type "no" to skip the
creation of Stackdriver account and terminate the deployment.
------------------------------------------------------------------------------
`, project.ID)
	log.Println(message)

	// Keep trying until Stackdriver account is ready, or user skips.
	for {
		ok, err := askForConfirmation()
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("user skipped the creation of Stackdriver account")
		}
		exist, err := stackdriverAccountExists(project.ID, rn)
		if err != nil {
			return err
		}
		if exist {
			log.Println("Stackdriver account has been created.")
			break
		}
		log.Println(`
------------------------------------------------------------------------------
The account is not created yet. It can take several minutes for it to be created.

After the account is created, enter [yes] to continue, or enter [no] to skip the
creation of Stackdriver account and terminate the deployment.
------------------------------------------------------------------------------
`)
	}
	return nil
}

// stackdriverAccountExists checks whether a Stackdriver account exists in the project.
func stackdriverAccountExists(projectID string, rn runner.Runner) (bool, error) {
	cmd := exec.Command("gcloud", "--project", projectID, "alpha", "monitoring", "policies", "list")
	out, err := rn.CmdCombinedOutput(cmd)
	if err != nil {
		if strings.Contains(string(out), "is not a workspace") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check stackdriver account existence: %v [%s]", err, string(out))
	}
	return true, nil
}

// askForConfirmation prompts the user to answer yes or no for confirmation.
func askForConfirmation() (bool, error) {
	var resp string
	if _, err := fmt.Scan(&resp); err != nil {
		return false, fmt.Errorf("failed to get user input: %v", err)
	}
	switch resp {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	default:
		fmt.Println("Please type [yes] or [no] and then press enter:")
		return askForConfirmation()
	}
}
