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
	"os"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
)

func defaultTerraform(config *config.Config, project *config.Project) error {
	if err := stateBucket(config, project); err != nil {
		return fmt.Errorf("failed to deploy terraform state: %v", err)
	}
	if err := dataResources(config, project, project.TerraformConfig.StateBucket); err != nil {
		return fmt.Errorf("failed to deploy terraform data resources: %v", err)
	}
	return nil
}

func stateBucket(config *config.Config, project *config.Project) error {
	if project.TerraformConfig == nil {
		return errors.New("terraform block in project must be set when terraform is enabled")
	}

	tfConf := terraform.NewConfig()
	opts := &terraform.Options{}
	addResources(tfConf, opts, project.TerraformConfig.StateBucket)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	return terraformApply(tfConf, dir, opts)
}

func dataResources(config *config.Config, project *config.Project, state *tfconfig.StorageBucket) error {
	rs := project.TerraformResources()
	if len(rs) == 0 {
		return nil
	}
	tfConf := terraform.NewConfig()
	tfConf.Terraform.Backend = &terraform.Backend{
		Bucket: state.Name,
		Prefix: "resources",
	}
	opts := &terraform.Options{}
	addResources(tfConf, opts, rs...)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	return terraformApply(tfConf, dir, opts)
}

func addResources(config *terraform.Config, opts *terraform.Options, resources ...tfconfig.Resource) {
	for _, r := range resources {
		config.Resources = append(config.Resources, &terraform.Resource{
			Name:       r.ID(),
			Type:       r.ResourceType(),
			Properties: r,
		})

		type importer interface {
			ImportID() string
		}

		if i, ok := r.(importer); ok {
			opts.Imports = append(opts.Imports, terraform.Import{
				Address: fmt.Sprintf("%s.%s", r.ResourceType(), r.ID()),
				ID:      i.ImportID(),
			})
		}

		type depender interface {
			DependentResources() []tfconfig.Resource
		}
		if d, ok := r.(depender); ok {
			addResources(config, opts, d.DependentResources()...)
		}
	}
}
