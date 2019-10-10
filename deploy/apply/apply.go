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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/deploymentmanager"
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

// Default applies project configurations to a default project.
func Default(conf *config.Config, project *config.Project, opts *Options) error {
	if err := verifyOrCreateProject(conf, project); err != nil {
		return fmt.Errorf("failed to verify or create project: %v", err)
	}

	if err := setupBilling(project, conf.Overall.BillingAccount); err != nil {
		return fmt.Errorf("failed to set up billing: %v", err)
	}

	if err := enableServiceAPIs(project); err != nil {
		return fmt.Errorf("failed to enable service APIs: %v", err)
	}

	if err := createCustomComputeImages(project); err != nil {
		return fmt.Errorf("failed to create compute images: %v", err)
	}

	if err := createDeletionLien(project); err != nil {
		return fmt.Errorf("failed to create deletion lien: %v", err)
	}

	if err := createStackdriverAccount(project); err != nil {
		return fmt.Errorf("failed to create stackdriver account: %v", err)
	}

	if err := DeployResources(conf, project, opts); err != nil {
		return fmt.Errorf("failed to deploy resources: %v", err)
	}

	if err := createAlerts(project); err != nil {
		return fmt.Errorf("failed to create alerts: %v", err)
	}

	if err := collectGCEInfo(project); err != nil {
		return fmt.Errorf("failed to collect GCE instances info: %v", err)
	}

	if fsa := conf.AllGeneratedFields.Forseti.ServiceAccount; fsa != "" {
		if err := GrantForsetiPermissions(project.ID, fsa); err != nil {
			return fmt.Errorf("failed to grant forseti access: %v", err)
		}
	}
	return nil
}

// DeployResources deploys the CFT resources in the project.
func DeployResources(conf *config.Config, project *config.Project, opts *Options) error {
	if !opts.DryRun {
		if err := grantDeploymentManagerAccess(project); err != nil {
			return fmt.Errorf("failed to grant deployment manager access to the project: %v", err)
		}
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
	sinkSA, err := getLogSinkServiceAccount(project, project.BQLogSink.Name())
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
		// Replace all instances of old writer SA with new.
		for _, a := range project.AuditLogs.LogsBQDataset.Accesses {
			if a.UserByEmail == currSA {
				a.UserByEmail = sinkSA
			}
		}
	}
	project.GeneratedFields.LogSinkServiceAccount = sinkSA

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
	serviceAcct := fmt.Sprintf("%s@cloudservices.gserviceaccount.com", pnum)

	// TODO: account for this in the rule generator.
	for _, role := range deploymentManagerRoles {
		if err := addBinding(project.ID, serviceAcct, role); err != nil {
			return fmt.Errorf("failed to grant role %q to DM service account %q: %v", role, serviceAcct, err)
		}
	}
	return nil
}

// addBinding adds an IAM policy binding for the given service account for the given role.
func addBinding(projectID, serviceAccount, role string) error {
	cmd := exec.Command(
		"gcloud", "projects", "add-iam-policy-binding", projectID,
		"--member", fmt.Sprintf("serviceAccount:%s", serviceAccount),
		"--role", fmt.Sprintf("roles/%s", role),
		"--project", projectID,
	)
	if err := runner.CmdRun(cmd); err != nil {
		return fmt.Errorf("failed to add iam policy binding for service account %q for role %q: %v", serviceAccount, role, err)
	}
	return nil
}

func getLogSinkServiceAccount(project *config.Project, sinkName string) (string, error) {
	cmd := exec.Command("gcloud", "logging", "sinks", "describe", sinkName, "--format", "json", "--project", project.ID)

	out, err := runner.CmdOutput(cmd)
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
	out, err := runner.CmdOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to get currently authenticated user: %v", err)
	}
	var member string
	if err := json.Unmarshal(out, &member); err != nil {
		return fmt.Errorf("failed to unmarshal current user: %v", err)
	}
	role := "roles/owner"
	member = "user:" + member

	// TODO: DM specific code. Remove once deployment manager has been deprecated.
	for _, p := range project.Resources.IAMPolicies {
		for _, b := range p.Bindings {
			if b.Role != role {
				continue
			}
			for _, m := range b.Members {
				if m == member {
					// User owner specifically requested, so don't remove them.
					return nil
				}
			}
		}
	}
	if project.IAMMembers != nil {
		for _, m := range project.IAMMembers.Members {
			if m.Role == role && m.Member == member {
				// User owner specifically requested, so don't remove them.
				return nil
			}
		}
	}

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
	return runner.CmdRun(cmd)
}

func hasBinding(project *config.Project, role string, member string) (has bool, err error) {
	cmd := exec.Command(
		"gcloud", "projects", "get-iam-policy", project.ID,
		"--project", project.ID,
		"--format", "json",
	)
	out, err := runner.CmdOutput(cmd)
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
			TmplPath:  "templates/audit_log_config.py",
		},
		&config.DefaultResource{
			OuterName: "chc-type-provider",
			TmplPath:  "templates/chc_resource/chc_res_type_provider.jinja",
		},
	}
	deployment, err := getDeployment(project, resources)
	if err != nil {
		return fmt.Errorf("failed to get deployment for pre-requisites: %v", err)
	}
	return upsertDeployment(setupPrerequisiteDeploymentName, deployment, project.ID)
}

func collectGCEInfo(project *config.Project) error {
	if len(project.Resources.GCEInstances) == 0 {
		project.GeneratedFields.GCEInstanceInfoList = nil
		return nil
	}
	cmd := exec.Command("gcloud", "--project", project.ID, "compute", "instances", "list", "--format", "json")
	out, err := runner.CmdOutput(cmd)
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

// verifyOrCreateProject verifies the project if exists or creates the project if does not exist.
//
// In the case where project exists, it needs to be ACTIVE and has the same organization ID or
// folder ID as those specified in the project config, if any. If project number is present
// in the generated fields, it also checks if the project ID corresponds to the project number.
// In the future, maybe consider changing the folder ID or organization ID of the existing project
// if different from config.
func verifyOrCreateProject(conf *config.Config, project *config.Project) error {
	orgID := conf.Overall.OrganizationID
	folderID := conf.Overall.FolderID
	if project.FolderID != "" {
		folderID = project.FolderID
	}

	var parentType, parentID string
	if folderID != "" {
		parentType = "folder"
		parentID = folderID
	} else if orgID != "" {
		parentType = "organization"
		parentID = orgID
	}

	// Enforce a check on the existing project number in the generated fields.
	pnum, err := verifyProject(project.ID, project.GeneratedFields.ProjectNumber, parentType, parentID)
	if err != nil {
		return err
	}
	if pnum != "" {
		project.GeneratedFields.ProjectNumber = pnum
		log.Printf("Project %q exists, skipping project creation.", project.ID)
		return nil
	}

	args := []string{"projects", "create", project.ID}
	if parentType == "" {
		log.Println("Creating project without a parent organization or folder.")
	} else {
		args = append(args, fmt.Sprintf("--%s", parentType), parentID)
	}

	cmd := exec.Command("gcloud", args...)
	if err := runner.CmdRun(cmd); err != nil {
		return fmt.Errorf("failed to run project creating command: %v", err)
	}
	pnum, err = verifyProject(project.ID, "", parentType, parentID)
	if err != nil {
		return fmt.Errorf("failed to verify newly created project: %v", err)
	}
	project.GeneratedFields.ProjectNumber = pnum

	return nil
}

// verifyProject checks project existence and enforces project config metadata.
// It returns the project number if exists and error if any.
func verifyProject(projectID, projectNumber, parentType, parentID string) (string, error) {
	cmd := exec.Command("gcloud", "projects", "describe", projectID, "--format", "json")
	out, err := runner.CmdOutput(cmd)
	if err != nil {
		// `gcloud projects describe` command might fail due to reasons other than project does not
		// exist (e.g. caller does not have sufficient permission). In that case, project could exist
		// and the code will return project existence as false. The caller might still attempt to create
		// the project and fail if the project already exists.
		log.Printf("Project %q does not exist or the caller does not have permission to see it. Attempting to create it...", projectID)
		return "", nil
	}

	// Project exists.
	type resourceID struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	type projectInfo struct {
		ProjectNumber  string     `json:"projectNumber"`
		LifecycleState string     `json:"lifecycleState"`
		Parent         resourceID `json:"parent"`
	}

	var pi projectInfo
	if err := json.Unmarshal(out, &pi); err != nil {
		return "", fmt.Errorf("failed to unmarshal project info output: %v", err)
	}

	if pi.ProjectNumber == "" {
		return "", fmt.Errorf("got empty project number: %v", err)
	}

	// Enforce check on the input project number if not empty.
	wantProjectNumber := pi.ProjectNumber
	if projectNumber != "" {
		wantProjectNumber = projectNumber
	}
	wantInfo := projectInfo{
		ProjectNumber:  wantProjectNumber,
		LifecycleState: "ACTIVE",
		Parent: resourceID{
			Type: parentType,
			ID:   parentID,
		},
	}

	if pi != wantInfo {
		return "", fmt.Errorf("project exists but has unexpected metadata: got %+v, want %+v", pi, wantInfo)
	}
	return pi.ProjectNumber, nil
}

// setupBilling sets the billing account for the project.
func setupBilling(project *config.Project, defaultBillingAccount string) error {
	ba := defaultBillingAccount
	if project.BillingAccount != "" {
		ba = project.BillingAccount
	}

	cmd := exec.Command("gcloud", "beta", "billing", "projects", "link", project.ID, "--billing-account", ba)
	if err := runner.CmdRun(cmd); err != nil {
		return fmt.Errorf("failed to link project to billing account %q: %v", ba, err)
	}
	return nil
}

// enableServiceAPIs enables service APIs for this project.
// Use this function instead of enabling private APIs in deployment manager because deployment
// management does not have all the APIs' access, which might triger PERMISSION_DENIED errors.
func enableServiceAPIs(project *config.Project) error {
	m := make(map[string]bool)
	for _, a := range project.EnabledAPIs {
		m[a] = true
	}
	m["deploymentmanager.googleapis.com"] = true
	// For project level iam policy updates.
	m["cloudresourcemanager.googleapis.com"] = true

	// TODO long term solution for updating APIs.
	if len(project.Resources.GCEInstances) > 0 {
		m["compute.googleapis.com"] = true
	}
	if len(project.Resources.CHCDatasets) > 0 {
		m["healthcare.googleapis.com"] = true
	}
	if len(project.Resources.GKEClusters) > 0 {
		m["container.googleapis.com"] = true
	}
	if len(project.Resources.IAMPolicies) > 0 || len(project.Resources.IAMCustomRoles) > 0 {
		m["iam.googleapis.com"] = true
	}

	var wantAPIs []string
	for a := range m {
		wantAPIs = append(wantAPIs, a)
	}
	sort.Slice(wantAPIs, func(i, j int) bool { return wantAPIs[i] < wantAPIs[j] })

	min := func(x, y int) int {
		if x < y {
			return x
		}
		return y
	}

	// Send in batches to avoid hitting quota limits.
	batchN := 10
	for i := 0; i < len(wantAPIs); i += batchN {
		args := []string{"--project", project.ID, "services", "enable"}
		args = append(args, wantAPIs[i:min(i+batchN, len(wantAPIs))]...)
		cmd := exec.Command("gcloud", args...)
		if err := runner.CmdRun(cmd); err != nil {
			return fmt.Errorf("failed to enable service APIs: %v", err)
		}
	}
	return nil
}

// createComputeImages creates new custom Compute Engine VM images, if specified.
// Create VM image using gcloud rather than deployment manager so that the deployment manager
// service account doesn't need to be granted access to the image GCS bucket.
// Note: for updates, only new images will be created. Existing images will not be modified.
// TODO: no longer need this after migrating to Terraform.
func createCustomComputeImages(project *config.Project) error {
	if len(project.Resources.GCEInstances) == 0 {
		log.Println("No GCE images to create.")
		return nil
	}
	for _, i := range project.Resources.GCEInstances {
		if i.CustomBootImage == nil {
			continue
		}
		// Check if custom image already exists.
		cmd := exec.Command("gcloud", "--project", project.ID, "compute", "images", "list",
			"--no-standard-images", "--filter", fmt.Sprintf("name=%s", i.CustomBootImage.ImageName), "--format", "value(name)")
		out, err := runner.CmdOutput(cmd)
		if err != nil {
			return fmt.Errorf("failed to check the existence of custom image %q: %v", i.CustomBootImage.ImageName, err)
		}
		if len(bytes.TrimSpace(out)) != 0 {
			log.Printf("Custom image %q already exists, skipping image creation.", i.CustomBootImage.ImageName)
			continue
		}
		// Create the image.
		cmd = exec.Command("gcloud", "--project", project.ID, "compute", "images", "create", i.CustomBootImage.ImageName, "--source-uri", fmt.Sprintf("gs://%s", i.CustomBootImage.GCSPath))
		if err := runner.CmdRun(cmd); err != nil {
			return fmt.Errorf("failed to create custom image %q: %v", i.CustomBootImage.ImageName, err)
		}
	}
	return nil
}

// createDeletionLien create the project deletion lien, if specified.
func createDeletionLien(project *config.Project) error {
	if !project.CreateDeletionLien {
		return nil
	}

	defaultLien := "resourcemanager.projects.delete"
	cmd := exec.Command("gcloud", "--project", project.ID, "alpha", "resource-manager", "liens",
		"list", "--filter", fmt.Sprintf("restrictions=%s", defaultLien), "--format", "value(restrictions)")
	out, err := runner.CmdOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to check existing deletion liens: %v", err)
	}
	if len(bytes.TrimSpace(out)) != 0 {
		log.Printf("Restriction lien %q already exists, skipping lien creation.", defaultLien)
		return nil
	}
	// Create the lien.
	cmd = exec.Command("gcloud", "--project", project.ID, "alpha", "resource-manager", "liens",
		"create", "--restrictions", defaultLien, "--reason", "Automated project deletion lien deployment.")
	if err := runner.CmdRun(cmd); err != nil {
		return fmt.Errorf("failed to create restriction lien %q: %v", defaultLien, err)
	}
	return nil
}

// TODO use Terraform once https://github.com/terraform-providers/terraform-provider-google/issues/2605 is resolved.
// createStackdriverAccount prompts the user to create a new Stackdriver Account.
func createStackdriverAccount(project *config.Project) error {
	if project.StackdriverAlertEmail == "" && len(project.NotificationChannels) == 0 {
		log.Println("No Stackdriver alert email or notification channels specified, skipping creation of Stackdriver account.")
		return nil
	}
	exist, err := stackdriverAccountExists(project.ID)
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
		exist, err := stackdriverAccountExists(project.ID)
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
func stackdriverAccountExists(projectID string) (bool, error) {
	cmd := exec.Command("gcloud", "--project", projectID, "alpha", "monitoring", "policies", "list")
	out, err := runner.CmdCombinedOutput(cmd)
	if err != nil {
		if strings.Contains(string(out), "not a Stackdriver workspace") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check stackdriver account existence: %v [%s]", err, string(out))
	}
	return true, nil
}

// createAlerts creates Stackdriver alerts for logs-based metrics.
// TODO: no longer need this after migrating to Terraform.
func createAlerts(project *config.Project) error {
	if project.StackdriverAlertEmail == "" {
		log.Println("No Stackdriver alert email specified, skipping creation of Stackdriver alerts.")
		return nil
	}

	type labels struct {
		EmailAddress string `json:"email_address"`
	}

	type channel struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		Type        string `json:"type"`
		Labels      labels `json:"labels"`
	}

	// Check channel existence and create if not.
	cmd := exec.Command("gcloud", "--project", project.ID, "alpha", "monitoring", "channels", "list",
		"--format", "json")

	out, err := runner.CmdOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to list existing monitoring channels: %v", err)
	}

	var channels []channel
	if err := json.Unmarshal(out, &channels); err != nil {
		return fmt.Errorf("failed to unmarshal exist monitoring channels list output: %v", err)
	}

	var exists bool
	var chanName string
	for _, c := range channels {
		// Assume only one channel exists per email.
		if c.Labels.EmailAddress == project.StackdriverAlertEmail {
			exists = true
			chanName = c.Name
			break
		}
	}
	if exists {
		log.Printf("Stackdriver notification channel already exists for %s.", project.StackdriverAlertEmail)
	} else {
		log.Println("Creating Stackdriver notification channel.")
		newChannel := channel{
			DisplayName: "Email",
			Type:        "email",
			Labels: labels{
				EmailAddress: project.StackdriverAlertEmail,
			},
		}

		b, err := json.Marshal(newChannel)
		if err != nil {
			return fmt.Errorf("failed to marshal channel to create: %v", err)
		}

		cmd := exec.Command("gcloud", "--project", project.ID, "alpha", "monitoring", "channels", "create",
			fmt.Sprintf("--channel-content=%s", string(b)), "--format", "json")
		out, err := runner.CmdOutput(cmd)
		if err != nil {
			return fmt.Errorf("failed to create new monitoring channel: %v", err)
		}
		var c channel
		if err := json.Unmarshal(out, &c); err != nil {
			return fmt.Errorf("failed to unmarshal created monitoring channel output: %v", err)
		}
		chanName = c.Name
	}

	if chanName == "" {
		return errors.New("channel name is empty")
	}

	// Create alerts.
	log.Printf("Creating Stackdriver alerts for channel %s.", chanName)

	// https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies
	type documentation struct {
		Content  string `json:"content"`
		MimeType string `json:"mimeType"`
	}
	type conditionThreshold struct {
		Comparison string `json:"comparison"`
		Filter     string `json:"filter"`
		Duration   string `json:"duration"`
	}
	type condition struct {
		Name               string              `json:"name"`
		DisplayName        string              `json:"displayName"`
		ConditionThreshold *conditionThreshold `json:"conditionThreshold"`
	}
	type alert struct {
		Name                 string         `json:"name"`
		DisplayName          string         `json:"displayName"`
		Documentation        *documentation `json:"documentation"`
		Conditions           []*condition   `json:"conditions"`
		Combiner             string         `json:"combiner"`
		Enabled              bool           `json:"enabled"`
		NotificationChannels []string       `json:"notificationChannels"`
	}

	// Get existing alerts.
	cmd = exec.Command("gcloud", "--project", project.ID, "alpha", "monitoring", "policies", "list", "--format", "json")

	out, err = runner.CmdOutput(cmd)
	if err != nil {
		return fmt.Errorf("failed to list existing monitoring alert policies: %v", err)
	}

	var alerts []alert
	if err := json.Unmarshal(out, &alerts); err != nil {
		return fmt.Errorf("failed to unmarshal exist monitoring channels list output: %v", err)
	}

	var existingAlerts = make(map[string]*alert)
	for _, a := range alerts {
		existingAlerts[a.DisplayName] = &a
	}

	// Default alerts are based on default custom logging metrics created in config.addBaseResources().
	alertsToCreate := []*alert{
		{
			DisplayName: "IAM Policy Change Alert",
			Documentation: &documentation{
				Content: "This policy ensures the designated user/group is notified when IAM policies are altered.",
			},
			Conditions: []*condition{
				{
					ConditionThreshold: &conditionThreshold{
						Filter: fmt.Sprintf(`resource.type=one_of("global","pubsub_topic","pubsub_subscription","gce_instance") AND metric.type="logging.googleapis.com/user/%s"`, config.IAMChangeMetricName),
					},
					DisplayName: fmt.Sprintf("No tolerance on %s!", config.IAMChangeMetricName),
				},
			},
		},
		{
			DisplayName: "Bucket Permission Change Alert",
			Documentation: &documentation{
				Content: "This policy ensures the designated user/group is notified when bucket/object permissions are altered.",
			},
			Conditions: []*condition{
				{
					ConditionThreshold: &conditionThreshold{
						Filter: fmt.Sprintf(`resource.type="gcs_bucket" AND metric.type="logging.googleapis.com/user/%s"`, config.BucketPermissionChangeMetricName),
					},
					DisplayName: fmt.Sprintf("No tolerance on %s!", config.BucketPermissionChangeMetricName),
				},
			},
		},
		{
			DisplayName: "Bigquery Update Alert",
			Documentation: &documentation{
				Content: "This policy ensures the designated user/group is notified when Bigquery dataset settings are altered.",
			},
			Conditions: []*condition{
				{
					ConditionThreshold: &conditionThreshold{
						Filter: fmt.Sprintf(`resource.type="global" AND metric.type="logging.googleapis.com/user/%s"`, config.BQSettingChangeMetricName),
					},
					DisplayName: fmt.Sprintf("No tolerance on %s!", config.BQSettingChangeMetricName),
				},
			},
		},
	}

	for _, b := range project.Resources.GCSBuckets {
		if len(b.ExpectedUsers) == 0 {
			continue
		}
		metricName := config.BucketUnexpectedAccessMetricPrefix + b.Name()
		alertsToCreate = append(alertsToCreate, &alert{
			DisplayName: fmt.Sprintf("Unexpected Access to %s Alert", b.Name()),
			Documentation: &documentation{
				Content: fmt.Sprintf("This policy ensures the designated user/group is notified when bucket %s is accessed by an unexpected user.", b.Name()),
			},
			Conditions: []*condition{
				{
					ConditionThreshold: &conditionThreshold{
						Filter: fmt.Sprintf(`resource.type="gcs_bucket" AND metric.type="logging.googleapis.com/user/%s"`, metricName),
					},
					DisplayName: fmt.Sprintf("No tolerance on %s!", metricName),
				},
			},
		})
	}

	for _, a := range alertsToCreate {
		if existingAlerts[a.DisplayName] != nil {
			continue
		}
		// Set common default values.
		a.Documentation.MimeType = "text/markdown"
		a.Combiner = "AND"
		a.Enabled = true
		a.NotificationChannels = []string{chanName}
		for _, c := range a.Conditions {
			c.ConditionThreshold.Comparison = "COMPARISON_GT"
			c.ConditionThreshold.Duration = "0s"
		}
		// Create alerts.
		log.Printf("Creating alert %q", a.DisplayName)
		b, err := json.Marshal(a)
		if err != nil {
			return fmt.Errorf("failed to marshal alert policy to create: %v", err)
		}
		cmd := exec.Command("gcloud", "--project", project.ID, "alpha", "monitoring", "policies", "create", fmt.Sprintf("--policy=%s", string(b)))
		if err := runner.CmdRun(cmd); err != nil {
			return fmt.Errorf("failed to create new alert policy %q: %v", a.DisplayName, err)
		}
	}

	return nil
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
