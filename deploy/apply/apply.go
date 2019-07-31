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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/deploymentmanager"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
)

const (
	deploymentNamePrefix            = "data-protect-toolkit"
	auditDeploymentName             = deploymentNamePrefix + "-audit"
	resourceDeploymentName          = deploymentNamePrefix + "-resources"
	setupPrerequisiteDeploymentName = deploymentNamePrefix + "-prerequisites"
)

// deploymentManagerRoles are the roles granted to the DM service account.
var deploymentManagerRoles = []string{"roles/owner", "roles/storage.admin"}

// deploymentRetryWaitTime is the time to wait between retrying a deployment to allow for concurrent operations to finish.
const deploymentRetryWaitTime = time.Minute

// The following vars are stubbed in tests.
var (
	cmdOutput = func(cmd *exec.Cmd) ([]byte, error) {
		log.Printf("Running: %v", cmd.Args)
		cmd.Stderr = os.Stderr
		return cmd.Output()
	}
	cmdRun = func(cmd *exec.Cmd) error {
		log.Printf("Running: %v", cmd.Args)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	upsertDeployment = deploymentmanager.Upsert
	terraformApply   = terraform.Apply
)

// deploymentManagerTyper should be implemented by resources that are natively supported by the deployment manager service.
// Use this if there is no suitable CFT template for a resource and a custom template is not needed.
// See https://cloud.google.com/deployment-manager/docs/configuration/supported-resource-types for valid types.
type deploymentManagerTyper interface {
	DeploymentManagerType() string
}

// deploymentManagerPather should be implemented by resources that use a DM template to deploy.
// Use this if the resource wraps a CFT or custom template.
type deploymentManagerPather interface {
	TemplatePath() string
}

// depender is the interface that defines a method to get dependent resources.
type depender interface {
	// Dependencies returns the name of the resource IDs to depend on.
	Dependencies() []string
}

// Apply deploys the CFT resources in the project.
func Apply(conf *config.Config, project *config.Project, opts *Options) error {
	if err := grantDeploymentManagerAccess(project); err != nil {
		return fmt.Errorf("failed to grant deployment manager access to the project: %v", err)
	}

	if err := deployPrerequisite(project); err != nil {
		return fmt.Errorf("failed to deploy pre-requisites: %v", err)
	}

	if err := importBinauthz(project.ID, project.BinauthzPolicy); err != nil {
		return fmt.Errorf("failed to import binary authorization policy: %v", err)
	}

	if err := deployResources(project); err != nil {
		return fmt.Errorf("failed to deploy resources: %v", err)
	}

	// Always get the latest log sink writer as when the sink is moved between deployments it may
	// create a new sink writer.
	sinkSA, err := getLogSinkServiceAccount(project)
	if err != nil {
		return fmt.Errorf("failed to get log sink service account: %v", err)
	}

	// Note: if the project was previously deployed, project.Init will already have set the log sink service account permission on the dataset.
	// An empty currSA implies this is the first time the sink was deployed.
	if currSA := project.GeneratedFields.LogSinkServiceAccount; currSA == "" {
		project.AuditLogs.LogsBQDataset.Accesses = append(project.AuditLogs.LogsBQDataset.Accesses, &config.Access{
			Role: "WRITER", UserByEmail: sinkSA,
		})
	} else if currSA != sinkSA {
		project.GeneratedFields.LogSinkServiceAccount = sinkSA
		// Replace all instances of old writer SA with new.
		for _, a := range project.AuditLogs.LogsBQDataset.Accesses {
			if a.UserByEmail == currSA {
				a.UserByEmail = sinkSA
			}
		}
	}

	if err := deployAudit(project, conf.ProjectForAuditLogs(project)); err != nil {
		return fmt.Errorf("failed to deploy audit resources: %v", err)
	}

	if err := deployGKEWorkloads(project); err != nil {
		return fmt.Errorf("failed to deploy GKE workloads: %v", err)
	}

	// Only remove owner account if there is an organization to ensure the project has an administrator.
	if conf.Overall.OrganizationID != "" {
		if err := removeOwnerUser(project); err != nil {
			return fmt.Errorf("failed to remove owner user: %v", err)
		}
	}
	return nil
}

// grantDeploymentManagerAccess grants the necessary permissions to the DM service account to perform its actions.
// Note: we don't revoke deployment manager's access because permissions can take up to 7 minutes
// to propagate through the system, which can cause permission denied issues when doing updates.
// This is not a problem on initial deployment since no resources have been created.
// DM is HIPAA compliant, so it's ok to leave its access.
// See https://cloud.google.com/iam/docs/granting-changing-revoking-access.
func grantDeploymentManagerAccess(project *config.Project) error {
	pnum := project.GeneratedFields.ProjectNumber
	if pnum == "" {
		return fmt.Errorf("project number not set in generated fields %+v", project.GeneratedFields)
	}
	serviceAcct := fmt.Sprintf("serviceAccount:%s@cloudservices.gserviceaccount.com", pnum)

	// TODO: account for this in the rule generator.
	for _, role := range deploymentManagerRoles {
		cmd := exec.Command(
			"gcloud", "projects", "add-iam-policy-binding", project.ID,
			"--role", role,
			"--member", serviceAcct,
			"--project", project.ID,
		)
		if err := cmdRun(cmd); err != nil {
			return fmt.Errorf("failed to grant role %q to DM service account %q: %v", role, serviceAcct, err)
		}
	}
	return nil
}

func getLogSinkServiceAccount(project *config.Project) (string, error) {
	cmd := exec.Command("gcloud", "logging", "sinks", "describe", project.BQLogSink.Name(), "--format", "json", "--project", project.ID)

	out, err := cmdOutput(cmd)
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

func deployAudit(project, auditProject *config.Project) error {
	rs := []config.Resource{&project.AuditLogs.LogsBQDataset}
	if project.AuditLogs.LogsGCSBucket != nil {
		rs = append(rs, project.AuditLogs.LogsGCSBucket)
	}
	deployment, err := getDeployment(project, rs)
	if err != nil {
		return err
	}

	// Append project ID to deployment name so each project has unique deployment if there is
	// a remote audit logs project.
	name := fmt.Sprintf("%s-%s", auditDeploymentName, project.ID)
	if err := upsertDeployment(name, deployment, auditProject.ID); err != nil {
		return fmt.Errorf("failed to deploy audit resources: %v", err)
	}
	return nil
}

func deployResources(project *config.Project) error {
	rs := project.DeploymentManagerResources()
	if len(rs) == 0 {
		log.Println("No resources to deploy.")
		return nil
	}
	deployment, err := getDeployment(project, rs)
	if err != nil {
		return err
	}
	if err := upsertDeployment(resourceDeploymentName, deployment, project.ID); err != nil {
		return fmt.Errorf("failed to deploy deployment manager resources: %v", err)
	}
	return nil
}

func getDeployment(project *config.Project, resources []config.Resource) (*deploymentmanager.Deployment, error) {
	deployment := &deploymentmanager.Deployment{}

	importSet := make(map[string]bool)

	for _, r := range resources {
		var typ string
		if typer, ok := r.(deploymentManagerTyper); ok {
			typ = typer.DeploymentManagerType()
		} else if pather, ok := r.(deploymentManagerPather); ok {
			var err error
			typ, err = filepath.Abs(pather.TemplatePath())
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for %q: %v", pather.TemplatePath(), err)
			}
			if !importSet[typ] {
				deployment.Imports = append(deployment.Imports, &deploymentmanager.Import{Path: typ})
				importSet[typ] = true
			}
		} else {
			return nil, fmt.Errorf("failed to get type of %+v", r)
		}

		b, err := json.Marshal(r)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource: %v", err)
		}

		type resourceProperties struct {
			Properties map[string]interface{} `json:"properties"`
		}
		rp := new(resourceProperties)
		if err := json.Unmarshal(b, &rp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource: %v", err)
		}

		res := &deploymentmanager.Resource{
			Name:       r.Name(),
			Type:       typ,
			Properties: rp.Properties,
		}

		if dr, ok := r.(depender); ok && len(dr.Dependencies()) > 0 {
			res.Metadata = &deploymentmanager.Metadata{DependsOn: dr.Dependencies()}
		}

		deployment.Resources = append(deployment.Resources, res)
	}

	return deployment, nil
}

func removeOwnerUser(project *config.Project) error {
	cmd := exec.Command("gcloud", "config", "get-value", "account", "--format", "json", "--project", project.ID)
	out, err := cmdOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to get currently authenticated user: %v", err)
	}
	var member string
	if err := json.Unmarshal(out, &member); err != nil {
		return fmt.Errorf("failed to unmarshal current user: %v", err)
	}
	role := "roles/owner"
	member = "user:" + member

	// TODO: check user specified bindings in case user wants the binding left
	has, err := hasBinding(project, role, member)
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
	return cmdRun(cmd)
}

func hasBinding(project *config.Project, role string, member string) (has bool, err error) {
	cmd := exec.Command(
		"gcloud", "projects", "get-iam-policy", project.ID,
		"--project", project.ID,
		"--format", "json",
	)
	out, err := cmdOutput(cmd)
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

// deployPrerequisite deploys the CHC resources in the project.
func deployPrerequisite(project *config.Project) error {
	resources := []config.Resource{
		&config.DefaultResource{
			OuterName: "enable-all-audit-log-policies",
			TmplPath:  "deploy/templates/audit_log_config.py",
		},
		&config.DefaultResource{
			OuterName: "chc-type-provider",
			TmplPath:  "deploy/templates/chc_resource/chc_res_type_provider.jinja",
		},
	}
	deployment, err := getDeployment(project, resources)
	if err != nil {
		return fmt.Errorf("failed to get deployment for pre-requisites: %v", err)
	}
	return upsertDeployment(setupPrerequisiteDeploymentName, deployment, project.ID)
}
