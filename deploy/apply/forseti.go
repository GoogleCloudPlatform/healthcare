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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

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
func Forseti(conf *config.Config, rn runner.Runner) error {
	project := conf.Forseti.Project
	if err := Default(conf, project, rn); err != nil {
		return err
	}

	// Always deploy state bucket, otherwise a forseti installation that failed half way through
	// will be left in a partial state and every following attempt will install a fresh instance.
	// TODO: once terraform is launched and default just let the Default take care of deploying the state bucket and remove this block.
	if err := stateBucket(project, rn); err != nil {
		return fmt.Errorf("failed to deploy terraform state: %v", err)
	}

	if err := forsetiConfig(conf, rn); err != nil {
		return fmt.Errorf("failed to apply forseti config: %v", err)
	}

	if err := GrantForsetiPermissions(project.ID, conf.AllGeneratedFields.Forseti.ServiceAccount, project.DevopsConfig.StateBucket.Name, rn); err != nil {
		return err
	}
	return nil
}

// forsetiConfig applies the forseti config, if it exists. It does not configure
// other settings such as billing account, deletion lien, etc.
func forsetiConfig(conf *config.Config, rn runner.Runner) error {
	if conf.Forseti == nil {
		log.Println("no forseti config, nothing to do")
		return nil
	}
	project := conf.Forseti.Project

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

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

	if err := terraformApply(tfConf, dir, nil, rn); err != nil {
		return err
	}

	serviceAccount, err := forsetiServerServiceAccount(project.ID, rn)
	if err != nil {
		return fmt.Errorf("failed to set Forseti server service account: %v", err)
	}
	conf.AllGeneratedFields.Forseti.ServiceAccount = serviceAccount

	serverBucket, err := forsetiServerBucket(project.ID, rn)
	if err != nil {
		return fmt.Errorf("failed to set Forseti server bucket: %v", err)
	}
	conf.AllGeneratedFields.Forseti.ServiceBucket = serverBucket
	return nil
}

func stateBucket(project *config.Project, rn runner.Runner) error {
	if project.DevopsConfig.StateBucket == nil {
		return errors.New("state_storage_bucket must not be nil")
	}

	tfConf := terraform.NewConfig()
	if err := addResources(tfConf, project.DevopsConfig.StateBucket); err != nil {
		return err
	}
	opts := &terraform.Options{}
	if err := addImports(opts, rn, project.DevopsConfig.StateBucket); err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	return terraformApply(tfConf, dir, opts, rn)
}

// GrantForsetiPermissions grants all necessary permissions to the given Forseti service account in the project.
func GrantForsetiPermissions(projectID, serviceAccount, stateBucket string, rn runner.Runner) error {
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
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	return terraformApply(tfConf, dir, &terraform.Options{}, rn)
}

// forsetiServerServiceAccount gets the server instance service account of the give Forseti project.
// TODO: Use Terraform state or output.
func forsetiServerServiceAccount(projectID string, rn runner.Runner) (string, error) {
	cmd := exec.Command(
		"gcloud", "--project", projectID,
		"iam", "service-accounts", "list", "--format", "json",
		"--filter", "email:forseti-server-gcp-*",
	)

	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to obtain Forseti server service account: %v", err)
	}

	type serviceAccount struct {
		Email string `json:"email"`
	}

	var serviceAccounts []serviceAccount
	if err := json.Unmarshal(out, &serviceAccounts); err != nil {
		return "", fmt.Errorf("failed to unmarshal service accounts output: %v", err)
	}
	if len(serviceAccounts) != 1 {
		return "", fmt.Errorf("unexpected number of Forseti server service accounts: got %d, want 1", len(serviceAccounts))
	}

	return serviceAccounts[0].Email, nil
}

// forsetiServerBucket gets the bucket holding the Forseti server instance's configuration.
// TODO: Use Terraform state or output.
func forsetiServerBucket(projectID string, rn runner.Runner) (string, error) {
	cmd := exec.Command("gsutil", "ls", "-p", projectID)

	out, err := rn.CmdOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to obtain Forseti server bucket: %v", err)
	}

	var bs []string
	for _, b := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(b, "gs://forseti-server-") {
			bs = append(bs, b)
		}
	}

	if len(bs) != 1 {
		return "", fmt.Errorf("unexpected number of Forseti server buckets: got %d, want 1", len(bs))
	}

	return bs[0], nil
}
