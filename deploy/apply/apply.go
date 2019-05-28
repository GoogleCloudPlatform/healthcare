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

	"github.com/GoogleCloudPlatform/healthcare/deploy/cft"
	"github.com/GoogleCloudPlatform/healthcare/deploy/deploymentmanager"
)

const (
	deploymentNamePrefix   = "data-protect-toolkit"
	auditDeploymentName    = deploymentNamePrefix + "-audit"
	resourceDeploymentName = deploymentNamePrefix + "-resources"
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
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	upsertDeployment = deploymentmanager.Upsert
)

// parsedResource is an interface that must be implemented by all concrete resource implementations.
type parsedResource interface {
	Init(*cft.Project) error
	Name() string
}

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
func Apply(config *cft.Config, project *cft.Project) error {
	if err := grantDeploymentManagerAccess(project); err != nil {
		return fmt.Errorf("failed to grant deployment manager access to the project: %v", err)
	}

	// TODO: stop retrying once
	// https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/issues/17
	// is fixed.
	for i := 0; i < 3; i++ {
		if err := deployResources(project); err == nil {
			break
		} else if i == 2 {
			return fmt.Errorf("failed to deploy resources: %v", err)
		}
		log.Printf("Sleeping for %v and retrying in case failure was due to concurrent IAM policy update", deploymentRetryWaitTime)
		time.Sleep(deploymentRetryWaitTime)
	}

	// If this is the initial deployment then there won't be a log sink service account
	// in the generated fields as it is deployed through the data-protect-toolkit-resources deployment.
	if project.GeneratedFields.LogSinkServiceAccount == "" {
		sinkSA, err := getLogSinkServiceAccount(project)
		if err != nil {
			return fmt.Errorf("failed to get log sink service account: %v", err)
		}
		project.GeneratedFields.LogSinkServiceAccount = sinkSA
		project.AuditLogs.LogsBQDataset.Accesses = append(project.AuditLogs.LogsBQDataset.Accesses, cft.Access{
			Role: "WRITER", UserByEmail: sinkSA,
		})
	}
	if err := deployAudit(project, config.ProjectForAuditLogs(project)); err != nil {
		return fmt.Errorf("failed to deploy audit resources: %v", err)
	}

	if err := deployGKEWorkloads(project); err != nil {
		return fmt.Errorf("failed to deploy GKE workloads: %v", err)
	}

	// Only remove owner account if there is an organization to ensure the project has an administrator.
	if config.Overall.OrganizationID != "" {
		if err := removeOwnerUser(project); err != nil {
			log.Printf("failed to remove owner user (might have already been removed): %v", err)
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
func grantDeploymentManagerAccess(project *cft.Project) error {
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

func getLogSinkServiceAccount(project *cft.Project) (string, error) {
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

func deployAudit(project, auditProject *cft.Project) error {
	pairs := []cft.ResourcePair{
		{Parsed: &project.AuditLogs.LogsBQDataset},
		{Parsed: project.AuditLogs.LogsGCSBucket},
	}
	deployment, err := getDeployment(project, pairs)
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

func deployResources(project *cft.Project) error {
	pairs := project.ResourcePairs()
	if len(pairs) == 0 {
		log.Println("No resources to deploy.")
		return nil
	}
	deployment, err := getDeployment(project, pairs)
	if err != nil {
		return err
	}
	if err := upsertDeployment(resourceDeploymentName, deployment, project.ID); err != nil {
		return fmt.Errorf("failed to deploy deployment manager resources: %v", err)
	}
	return nil
}

func getDeployment(project *cft.Project, pairs []cft.ResourcePair) (*deploymentmanager.Deployment, error) {
	deployment := &deploymentmanager.Deployment{}

	importSet := make(map[string]bool)

	for _, pair := range pairs {
		var typ string
		if typer, ok := pair.Parsed.(deploymentManagerTyper); ok {
			typ = typer.DeploymentManagerType()
		} else if pather, ok := pair.Parsed.(deploymentManagerPather); ok {
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
			return nil, fmt.Errorf("failed to get type of %+v", pair.Parsed)
		}

		merged, err := pair.MergedPropertiesMap()
		if err != nil {
			return nil, fmt.Errorf("failed to merge raw map with parsed: %v", err)
		}

		res := &deploymentmanager.Resource{
			Name:       pair.Parsed.Name(),
			Type:       typ,
			Properties: merged,
		}

		if dr, ok := pair.Parsed.(depender); ok && len(dr.Dependencies()) > 0 {
			res.Metadata = &deploymentmanager.Metadata{DependsOn: dr.Dependencies()}
		}

		deployment.Resources = append(deployment.Resources, res)
	}

	return deployment, nil
}

func removeOwnerUser(project *cft.Project) error {
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
		return nil
	}

	cmd = exec.Command(
		"gcloud", "projects", "remove-iam-policy-binding", project.ID,
		"--member", member, "--role", role, "--project", project.ID)
	return cmdRun(cmd)
}

func hasBinding(project *cft.Project, role string, member string) (has bool, err error) {
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
		Bindings []cft.Binding `json:"bindings"`
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
