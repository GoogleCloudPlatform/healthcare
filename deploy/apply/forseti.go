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
	"fmt"
	"log"
	"os/exec"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
)

// Standard (built in) roles required by the Forseti service account on projects to be monitored.
// This list includes project level roles from
// https://github.com/forseti-security/terraform-google-forseti/blob/master/modules/server/main.tf#L63
// In the future, have a deeper integration with Forseti module and reuse the role list.
var forsetiStandardRoles = [...]string{
	"appengine.appViewer",
	"bigquery.metadataViewer",
	"browser",
	"cloudasset.viewer",
	"cloudsql.viewer",
	"compute.networkViewer",
	"iam.securityReviewer",
	"servicemanagement.quotaViewer",
	"serviceusage.serviceUsageConsumer",
}

// Forseti applies project configuration to a Forseti project.
func Forseti(conf *config.Config, opts *Options, terraformConfigsDir string, rn runner.Runner) error {
	project := conf.Forseti.Project
	if err := Default(conf, project, opts, rn); err != nil {
		return err
	}
	workDir, err := terraform.WorkDir(terraformConfigsDir, project.ID)
	if err != nil {
		return err
	}
	// Always deploy state bucket, otherwise a forseti installation that failed half way through
	// will be left in a partial state and every following attempt will install a fresh instance.
	// TODO: once terraform is launched and default just let the Default take care of deploying the state bucket and remove this block.
	if err := stateBucket(project, opts, workDir, rn); err != nil {
		return fmt.Errorf("failed to deploy terraform state: %v", err)
	}

	if err := forsetiConfig(conf, opts, workDir, rn); err != nil {
		return fmt.Errorf("failed to apply forseti config: %v", err)
	}

	if err := GrantForsetiPermissions(project.ID, conf.AllGeneratedFields.Forseti.ServiceAccount, project.DevopsConfig.StateBucket.Name, opts, workDir, rn); err != nil {
		return err
	}
	return nil
}

// forsetiConfig applies the forseti config, if it exists. It does not configure
// other settings such as billing account, deletion lien, etc.
func forsetiConfig(conf *config.Config, opts *Options, workDir string, rn runner.Runner) error {
	if conf.Forseti == nil {
		log.Println("no forseti config, nothing to do")
		return nil
	}
	tfConf := terraform.NewConfig()
	tfConf.Modules = []*terraform.Module{{
		Name:       "forseti",
		Source:     "./external/terraform_google_forseti",
		Properties: conf.Forseti.Properties,
	}}

	tfConf.Terraform.Backend = &terraform.Backend{
		Bucket: conf.Forseti.Project.DevopsConfig.StateBucket.Name,
		Prefix: "forseti",
	}

	tfConf.Outputs = []*terraform.Output{
		{
			Name:  "forseti_server_service_account",
			Value: "${module.forseti.forseti-server-service-account}",
		},
		{
			Name:  "forseti_server_bucket",
			Value: "${module.forseti.forseti-server-storage-bucket}",
		},
	}

	workDir, err := terraform.WorkDir(workDir, "forseti")
	if err != nil {
		return err
	}
	if err := terraformApply(tfConf, workDir, &terraform.Options{ApplyFlags: opts.TerraformApplyFlags}, rn); err != nil {
		return err
	}

	serviceAccount, err := forsetiServerServiceAccount(workDir, rn)
	if err != nil {
		return fmt.Errorf("failed to set Forseti server service account: %v", err)
	}
	conf.AllGeneratedFields.Forseti.ServiceAccount = serviceAccount

	serverBucket, err := forsetiServerBucket(workDir, rn)
	if err != nil {
		return fmt.Errorf("failed to set Forseti server bucket: %v", err)
	}
	conf.AllGeneratedFields.Forseti.ServiceBucket = serverBucket
	return nil
}

// GrantForsetiPermissions grants all necessary permissions to the given Forseti service account in the project.
func GrantForsetiPermissions(projectID, serviceAccount, stateBucket string, opts *Options, workDir string, rn runner.Runner) error {
	iamMembers := new(tfconfig.ProjectIAMMembers)
	if err := iamMembers.Init(projectID); err != nil {
		return fmt.Errorf("failed to init IAM members resource: %v", err)
	}
	for _, r := range forsetiStandardRoles {
		iamMembers.Members = append(iamMembers.Members,
			&tfconfig.ProjectIAMMember{Role: "roles/" + r, Member: "serviceAccount:" + serviceAccount},
		)
	}

	tfConf := terraform.NewConfig()
	// TODO: Remove this check once TF is always enabled as the bucket should always be present then.
	if stateBucket != "" {
		tfConf.Terraform.Backend = &terraform.Backend{
			Bucket: stateBucket,
			Prefix: "forseti-access",
		}
	}

	if err := addResources(tfConf, iamMembers); err != nil {
		return err
	}

	workDir, err := terraform.WorkDir(workDir, "forseti-access")
	if err != nil {
		return err
	}

	return terraformApply(tfConf, workDir, &terraform.Options{ApplyFlags: opts.TerraformApplyFlags}, rn)
}

// forsetiServerServiceAccount gets the server instance service account of the give Forseti project.
func forsetiServerServiceAccount(dir string, rn runner.Runner) (string, error) {
	cmd := exec.Command("terraform", "output", "-json", "forseti_server_service_account")
	cmd.Dir = dir
	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get forseti-server-service-account Terraform output: %v", err)
	}
	var sa string
	if err := json.Unmarshal(out, &sa); err != nil {
		return "", fmt.Errorf("failed to parse Forseti server service account from terraform output: %v", err)
	}
	return sa, nil
}

// forsetiServerBucket gets the bucket holding the Forseti server instance's configuration.
func forsetiServerBucket(dir string, rn runner.Runner) (string, error) {
	cmd := exec.Command("terraform", "output", "-json", "forseti_server_bucket")
	cmd.Dir = dir
	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get forseti-server-storage-bucket Terraform output: %v", err)
	}
	var bn string
	if err := json.Unmarshal(out, &bn); err != nil {
		return "", fmt.Errorf("failed to parse Forseti server storage bucket from terraform output: %v", err)
	}
	return fmt.Sprintf("gs://%s/", bn), nil
}
