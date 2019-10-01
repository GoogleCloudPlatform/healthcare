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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
)

// Terraform applies the project configs for each applicable project in the config to GCP.
// Empty list of projects is equivalent to all projects.
// Base projects (remote audit and forseti) are deployed in a round-table fashion with each phase being applied to each project.
// This makes sure dependencies between base projects are handled correctly
// (e.g. forseti project wants to store its audit logs in remote audit project while remote audit project wants to be monitored by the forseti project).
// Then, all data hosting projects are deployed from beginning till end so one data project doesn't leave other data projects in a half deployed state.
func Terraform(conf *config.Config, projectIDs []string) error {
	idSet := make(map[string]bool)
	for _, p := range projectIDs {
		idSet[p] = true
	}

	wantProject := func(p *config.Project) bool {
		return len(projectIDs) == 0 || idSet[p.ID]
	}

	var baseProjects, dataProjects []*config.Project

	if wantProject(conf.AuditLogsProject) {
		baseProjects = append(baseProjects, conf.AuditLogsProject)
	}

	if conf.Forseti != nil && wantProject(conf.Forseti.Project) {
		baseProjects = append(baseProjects, conf.Forseti.Project)
	}

	for _, p := range conf.Projects {
		if wantProject(p) {
			dataProjects = append(dataProjects, p)
		}
	}

	// Apply all base projects in a round-table fashion.
	log.Println("Applying base projects")
	if err := projects(conf, baseProjects...); err != nil {
		return fmt.Errorf("failed to apply base projects: %v", err)
	}

	// Apply each data hosting project from beginning to end.
	for _, p := range dataProjects {
		log.Printf("Applying project %q", p.ID)
		if err := projects(conf, p); err != nil {
			return fmt.Errorf("failed to apply %q: %v", p.ID, err)
		}
	}
	return nil
}

// projects applies all phases in sequence to each project.
func projects(conf *config.Config, projs ...*config.Project) error {
	for _, p := range projs {
		log.Printf("Creating project and state bucket for %q", p.ID)
		if err := createProjectTerraform(conf, p); err != nil {
			return err
		}
	}

	for _, p := range projs {
		log.Printf("Appling project %q", p.ID)
		if err := services(p); err != nil {
			return fmt.Errorf("failed to apply services: %v", err)
		}
		if err := createStackdriverAccount(p); err != nil {
			return err
		}
		if err := defaultTerraform(conf, p); err != nil {
			return fmt.Errorf("failed to apply resources for %q: %v", p.ID, err)
		}
		if err := collectGCEInfo(p); err != nil {
			return fmt.Errorf("failed to collect GCE instances info: %v", err)
		}
	}

	for _, p := range projs {
		log.Printf("Applying audit resources for %q", p.ID)
		if err := auditResources(conf, p); err != nil {
			return fmt.Errorf("failed to apply audit resources for %q: %v", p.ID, err)
		}
	}

	if conf.Forseti == nil {
		return nil
	}

	for _, p := range projs {
		// Deploy the forseti instance if forseti project is being requested.
		if conf.Forseti.Project.ID == p.ID {
			log.Printf("Applying Forseti instance in %q", conf.Forseti.Project.ID)
			if err := forsetiConfig(conf); err != nil {
				return fmt.Errorf("failed to apply forseti instance in %q: %v", conf.Forseti.Project.ID, err)
			}
			break
		}
	}

	fsa := conf.AllGeneratedFields.Forseti.ServiceAccount
	if fsa == "" {
		return errors.New("forseti service account not set in generated fields")
	}
	for _, p := range projs {
		log.Printf("Granting Forseti instance access to project %q", p.ID)
		if err := GrantForsetiPermissions(p.ID, fsa); err != nil {
			return fmt.Errorf("failed to grant forseti access to project %q: %v", p.ID, err)
		}
	}
	return nil
}

func createProjectTerraform(config *config.Config, project *config.Project) error {
	fid := project.FolderID
	if fid == "" {
		fid = config.Overall.FolderID
	}

	// Only one of folder ID or org ID can be set in the project resource.
	var oid string
	if fid == "" {
		oid = config.Overall.OrganizationID
	}

	ba := project.BillingAccount
	if ba == "" {
		ba = config.Overall.BillingAccount
	}

	pr := &tfconfig.ProjectResource{
		OrgID:          oid,
		FolderID:       fid,
		BillingAccount: ba,
	}
	if err := pr.Init(project.ID); err != nil {
		return err
	}

	tfConf := terraform.NewConfig()
	opts := &terraform.Options{}
	if err := addResources(tfConf, opts, pr, project.TerraformConfig.StateBucket); err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	return terraformApply(tfConf, dir, opts)
}

func defaultTerraform(config *config.Config, project *config.Project) error {
	// TODO: merge services with resources.
	if err := services(project); err != nil {
		return fmt.Errorf("failed to apply services: %v", err)
	}

	if err := resources(project); err != nil {
		return fmt.Errorf("failed to apply resources: %v", err)
	}

	if err := auditResources(config, project); err != nil {
		return fmt.Errorf("failed to apply audit resources: %v", err)
	}

	// TODO: the user being used for terraform is actually set by gcloud auth application-default login.
	// This user can differ than the one set by gcloud auth login which is what removeOwnerUser checks for.
	// Fix this to remove the application-default owner.
	if err := removeOwnerUser(project); err != nil {
		return fmt.Errorf("failed to remove authenticated user: %v", err)
	}

	return nil
}

func services(project *config.Project) error {
	tfConf := terraform.NewConfig()
	tfConf.Terraform.Backend = &terraform.Backend{
		Bucket: project.TerraformConfig.StateBucket.Name,
		Prefix: "services",
	}

	opts := &terraform.Options{}
	if err := addResources(tfConf, opts, project.Services); err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	if err := terraformApply(tfConf, dir, opts); err != nil {
		return err
	}
	return nil
}

func auditResources(config *config.Config, project *config.Project) error {
	auditProject := config.ProjectForAuditLogs(project)
	tfConf := terraform.NewConfig()
	tfConf.Terraform.Backend = &terraform.Backend{
		Bucket: auditProject.TerraformConfig.StateBucket.Name,
		// Attach project ID to prefix.
		// This is because the audit project will contain an audit deployment for every project it holds audit logs for in its own state bucket.
		// Attaching the prefix ensures we don't have a collision in names.
		Prefix: "audit-" + project.ID,
	}

	d := project.Audit.LogsBigqueryDataset
	d.Accesses = append(d.Accesses, &tfconfig.Access{
		Role: "WRITER",

		// Trim "serviceAccount:" prefix as user_by_email does not allow it.
		UserByEmail: fmt.Sprintf(
			`${replace(google_logging_project_sink.%s.writer_identity, "serviceAccount:", "")}`,
			project.BQLogSinkTF.ID(),
		),
	})

	rs := []tfconfig.Resource{project.BQLogSinkTF, d}
	if b := project.Audit.LogsStorageBucket; b != nil {
		rs = append(rs, b)
	}

	opts := &terraform.Options{}
	if err := addResources(tfConf, opts, rs...); err != nil {
		return nil
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	if err := terraformApply(tfConf, dir, opts); err != nil {
		return err
	}

	project.GeneratedFields.LogSinkServiceAccount, err = getLogSinkServiceAccount(project, project.BQLogSinkTF.Name)
	if err != nil {
		return fmt.Errorf("failed to get log sink service account: %v", err)
	}
	return nil
}

func resources(project *config.Project) error {
	rs := project.TerraformResources()
	tfConf := terraform.NewConfig()

	// Needed to work around issues like https://github.com/terraform-providers/terraform-provider-google/issues/4460.
	// Also used if a resource does not explicitly set the project field.
	tfConf.Providers = append(tfConf.Providers,
		&terraform.Provider{
			Name:       "google",
			Properties: map[string]interface{}{"project": project.ID},
		},
		// Beta provider needed for some resources such as healthcare resources.
		&terraform.Provider{
			Name:       "google-beta",
			Properties: map[string]interface{}{"project": project.ID},
		},
	)

	tfConf.Terraform.Backend = &terraform.Backend{
		Bucket: project.TerraformConfig.StateBucket.Name,
		Prefix: "resources",
	}

	// Allow user resources to access project runtime info such as project number.
	tfConf.Data = []*terraform.Resource{{
		Name: project.ID,
		Type: "google_project",
		Properties: map[string]interface{}{
			"project_id": project.ID,
		},
	}}

	opts := &terraform.Options{}
	if err := addResources(tfConf, opts, rs...); err != nil {
		return err
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	return terraformApply(tfConf, dir, opts)
}

// addResources adds the given resources to the given terraform config.
// If the resource implements functions to get an import ID, the terraform options will be updated
// with the import ID, so the resource is imported to the terraform state if it already exists.
// The import ID for a resource can be found in terraform's documentation for the resource.
func addResources(config *terraform.Config, opts *terraform.Options, resources ...tfconfig.Resource) error {
	for _, r := range resources {
		config.Resources = append(config.Resources, &terraform.Resource{
			Name:       r.ID(),
			Type:       r.ResourceType(),
			Properties: r,
		})

		type importer interface {
			ImportID() (string, error)
		}

		if i, ok := r.(importer); ok {
			id, err := i.ImportID()
			if err != nil {
				return fmt.Errorf("failed to get import ID for %q %q: %v", r.ResourceType(), r.ID(), err)
			}
			if id != "" {
				opts.Imports = append(opts.Imports, terraform.Import{
					Address: fmt.Sprintf("%s.%s", r.ResourceType(), r.ID()),
					ID:      id,
				})
			}
		}

		type depender interface {
			DependentResources() []tfconfig.Resource
		}
		if d, ok := r.(depender); ok {
			if err := addResources(config, opts, d.DependentResources()...); err != nil {
				return err
			}
		}
	}
	return nil
}
