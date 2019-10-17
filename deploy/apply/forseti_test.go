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
	"testing"

	"github.com/GoogleCloudPlatform/healthcare/deploy/runner"
	"github.com/GoogleCloudPlatform/healthcare/deploy/terraform"
	"github.com/GoogleCloudPlatform/healthcare/deploy/testconf"
	"github.com/google/go-cmp/cmp"
)

func TestForsetiConfig(t *testing.T) {
	conf, _ := testconf.ConfigAndProject(t, nil)

	var gotTFConf *terraform.Config
	terraformApply = func(config *terraform.Config, _ string, _ *terraform.Options, _ runner.Runner) error {
		gotTFConf = config
		return nil
	}

	if err := forsetiConfig(conf, &runner.Fake{}); err != nil {
		t.Errorf("Forseti = %v", err)
	}

	wantConfig := `
terraform:
  required_version: ">= 0.12.0"
  backend:
    gcs:
      bucket: my-forseti-project-state
      prefix: forseti
module:
- forseti:
    source: "./external/terraform_google_forseti"
    composite_root_resources:
    - organizations/12345678
    - folders/98765321
    domain: my-domain.com
    project_id: my-forseti-project
    storage_bucket_location: us-east1
`

	b, err := json.Marshal(gotTFConf)
	if err != nil {
		t.Errorf("json.Marshal gotTFConf: %v", err)
	}
	if diff := cmp.Diff(unmarshal(t, string(b)), unmarshal(t, wantConfig)); diff != "" {
		t.Errorf("terraform config differs (-got, +want):\n%v", diff)
	}
}

func TestGrantForsetiPermissions(t *testing.T) {
	var gotTFConf *terraform.Config
	terraformApply = func(config *terraform.Config, _ string, _ *terraform.Options, _ runner.Runner) error {
		gotTFConf = config
		return nil
	}

	if err := GrantForsetiPermissions("project1", "forseti-sa@@forseti-project.iam.gserviceaccount.com", "my-forseti-project-state", &runner.Fake{}); err != nil {
		t.Errorf("GrantForsetiPermissions = %v", err)
	}

	wantConfig := `
terraform:
  required_version: ">= 0.12.0"
  backend:
    gcs:
      bucket: my-forseti-project-state
      prefix: forseti-access
resource:
- google_project_iam_member:
    project:
      role: "${each.value.role}"
      member: "${each.value.member}"
      for_each:
        roles/appengine.appViewer serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/appengine.appViewer
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
        roles/bigquery.metadataViewer serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/bigquery.metadataViewer
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
        roles/browser serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/browser
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
        roles/cloudasset.viewer serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/cloudasset.viewer
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
        roles/cloudsql.viewer serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/cloudsql.viewer
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
        roles/compute.networkViewer serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/compute.networkViewer
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
        roles/iam.securityReviewer serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/iam.securityReviewer
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
        roles/servicemanagement.quotaViewer serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/servicemanagement.quotaViewer
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
        roles/serviceusage.serviceUsageConsumer serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com:
          role: roles/serviceusage.serviceUsageConsumer
          member: serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com
      project: project1
`

	b, err := json.Marshal(gotTFConf)
	if err != nil {
		t.Errorf("json.Marshal gotTFConf: %v", err)
	}
	if diff := cmp.Diff(unmarshal(t, string(b)), unmarshal(t, wantConfig)); diff != "" {
		t.Errorf("terraform config differs (-got, +want):\n%v", diff)
	}
}
