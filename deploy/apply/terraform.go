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
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
)

func deployTerraform(config *config.Config, project *config.Project) error {
	if project.TerraformConfig == nil {
		return errors.New("terraform block in project must be set when terraform is enabled")
	}
	b := project.TerraformConfig.StateBucket

	tfConf := &terraform.Config{
		Resources: []*terraform.Resource{{
			Name:       b.Name(),
			Type:       b.TerraformResourceName(),
			Properties: b.StorageBucketProperties,
		}},
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	imports := []terraform.Import{{
		Address: "google_storage_bucket." + b.Name(),
		ID:      fmt.Sprintf("%s/%s", project.ID, b.Name()),
	}}
	return terraformApply(tfConf, dir, &terraform.Options{Imports: imports})
}
