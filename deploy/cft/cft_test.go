package cft

import (
	"log"
	"os"
	"testing"

	"gopkg.in/yaml.v2"
)

const configYAML = `
overall:
  organization_id: '12345678'
  folder_id: '98765321'
  billing_account: 000000-000000-000000
  domain: 'domain.com'

projects:
- project_id: my-project
  owners_group: my-project-owners@my-domain.com
  editors_group: my-project-editors@mydomain.com
  auditors_group: some-auditors-group@my-domain.com
  data_readwrite_groups:
  - some-readwrite-group@my-domain.com
  data_readonly_groups:
  - some-readonly-group@my-domain.com
  - another-readonly-group@googlegroups.com
  audit_logs:
    logs_gcs_bucket:
      location: US
      storage_class: MULTI_REGIONAL
      ttl_days: 365
    logs_bigquery_dataset:
      location: US
  resources:
    bigquery_datasets:
    - cft:
        name: my-dataset
`

var (
	config  *Config
	project *Project
)

func TestDeploy(t *testing.T) {
	// TODO: implement CFT integration test once rest of the code is checked in.
}

func TestMain(m *testing.M) {
	config = new(Config)
	if err := yaml.Unmarshal([]byte(configYAML), config); err != nil {
		log.Fatalf("unmarshal config: %v", err)
	}
	if len(config.Projects) != 1 {
		log.Fatalf("len(config.Projects)=%v, want 1", len(config.Projects))
	}
	project = config.Projects[0]
	os.Exit(m.Run())
}
