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

// Package testconf provides utilities to create test configs.
package testconf

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/GoogleCloudPlatform/healthcare/deploy/config"
	"github.com/ghodss/yaml"
)

// TODO: add devops and remote audit project and add integration test that deploys all projects.
const configYAML = `
overall:
  organization_id: '12345678'
  folder_id: '98765321'
  billing_account: 000000-000000-000000
  domain: 'my-domain.com'
  allowed_apis:
  - foo-api.googleapis.com
  - bar-api.googleapis.com

forseti:
  project:
    project_id: my-forseti-project
    owners_group: my-forseti-project-owners@my-domain.com
    auditors_group: my-forseti-project-auditors@my-domain.com
    terraform:
      state_storage_bucket:
        name: my-forseti-project-state
        location: US
    audit_logs:
      logs_bq_dataset:
        properties:
          name: audit_logs
          location: US
      logs_gcs_bucket:
        ttl_days: 365
        properties:
          name: my-forseti-project-logs
          location: US
          storageClass: MULTI_REGIONAL
    audit:
      logs_bigquery_dataset:
        dataset_id: audit_logs
        location: US
      logs_storage_bucket:
        name: my-forseti-project-logs
        location: US
        storage_class: MULTI_REGIONAL
  properties:
    storage_bucket_location: us-east1

projects:
- project_id: my-project
  owners_group: my-project-owners@my-domain.com
  auditors_group: my-project-auditors@my-domain.com
  data_readwrite_groups:
  - my-project-readwrite@my-domain.com
  data_readonly_groups:
  - my-project-readonly@my-domain.com
  - another-readonly-group@googlegroups.com
  terraform:
    state_storage_bucket:
      name: my-project-state
      location: US
  enabled_apis:
  - foo-api.googleapis.com
  audit_logs:
    logs_bq_dataset:
      properties:
        name: audit_logs
        location: US
    logs_gcs_bucket:
      ttl_days: 365
      properties:
        name: my-project-logs
        location: US
        storageClass: MULTI_REGIONAL
  audit:
    logs_bigquery_dataset:
      dataset_id: audit_logs
      location: US
    logs_storage_bucket:
      name: my-project-logs
      location: US
      storage_class: MULTI_REGIONAL
{{lpad .ExtraProjectConfig 2}}
`

const generatedFieldsYAML = `
projects:
  my-project:
    project_number: '1111'
    log_sink_service_account: audit-logs-bq@logging-1111.iam.gserviceaccount.com
    gce_instance_info:
    - name: foo-instance
      id: '123'
  my-forseti-project:
    project_number: '2222'
    log_sink_service_account: audit-logs-bq@logging-2222.iam.gserviceaccount.com
    gce_instance_info:
    - name: foo-instance
      id: '123'
forseti:
  service_account: forseti@my-forseti-project.iam.gserviceaccount.com
  server_bucket: gs://my-forseti-project-server/
`

// ConfigData configures a config.
type ConfigData struct {
	ExtraProjectConfig string
}

// ConfigBeforeInit gets config that did not call the init() function.
func ConfigBeforeInit(t *testing.T, data *ConfigData) *config.Config {
	t.Helper()
	if data == nil {
		data = &ConfigData{}
	}

	tmpl, err := template.New("test").Funcs(template.FuncMap{"lpad": lpad}).Parse(configYAML)
	if err != nil {
		t.Fatalf("template Parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template Execute: %v", err)
	}

	if err := config.ValidateConf(buf.Bytes()); err != nil {
		t.Fatalf("validate config: %v", err)
	}

	conf := new(config.Config)
	if err := yaml.Unmarshal(buf.Bytes(), conf); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	return conf
}

// ConfigAndProject gets a test config and project.
func ConfigAndProject(t *testing.T, data *ConfigData) (*config.Config, *config.Project) {
	genFields := new(config.AllGeneratedFields)
	if err := yaml.Unmarshal([]byte(generatedFieldsYAML), genFields); err != nil {
		t.Fatalf("unmarshal generated fields: %v", err)
	}
	conf := ConfigBeforeInit(t, data)
	if err := conf.Init(genFields); err != nil {
		t.Fatalf("conf.Init = %v", err)
	}
	if len(conf.Projects) != 1 {
		t.Fatalf("len(conf.Projects)=%v, want 1", len(conf.Projects))
	}
	proj := conf.Projects[0]
	return conf, proj
}

func lpad(s string, n int) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		b.WriteString(strings.Repeat(" ", n))
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
