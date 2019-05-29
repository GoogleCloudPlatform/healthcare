package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

const globalBigqueryRule = `
- name: No public, domain or special group dataset access.
  mode: blacklist
  resource:
  - type: organization
    resource_ids: ['12345678']
  dataset_ids:
  - '*'
  bindings:
  - role: '*'
    members:
      - domain: '*'
      - special_group: '*'
- name: Whitelist for project my-forseti-project audit logs.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-forseti-project
  dataset_ids:
  - my-forseti-project:audit_logs
  bindings:
  - role: OWNER
    members:
    - group_email: my-forseti-project-owners@my-domain.com
  - role: WRITER
    members:
    - user_email: audit-logs-bq@logging-2222.iam.gserviceaccount.com
  - role: READER
    members:
    - group_email: my-forseti-project-auditors@my-domain.com
`

const auditDatasetRule = `
- name: Whitelist for project my-project audit logs.
  mode: whitelist
  resource:
  - type: project
    resource_ids:
     - my-project
  dataset_ids:
  - my-project:audit_logs
  bindings:
  - role: OWNER
    members:
    - group_email: my-project-owners@my-domain.com
  - role: WRITER
    members:
    - user_email: audit-logs-bq@logging-1111.iam.gserviceaccount.com
  - role: READER
    members:
    - group_email: my-project-auditors@my-domain.com`

func TestBigqueryRules(t *testing.T) {
	tests := []struct {
		name       string
		configData *ConfigData
		wantYAML   string
	}{
		{
			name: "global",
		},
		{
			name: "dataset",
			configData: &ConfigData{`
resources:
  bq_datasets:
  - properties:
      name: foo-dataset
      location: US
  - properties:
      name: bar-dataset
      location: US`},
			wantYAML: `
- name: 'Whitelist for dataset(s): my-project:foo-dataset, my-project:bar-dataset.'
  mode: whitelist
  resource:
  - type: project
    resource_ids:
    - my-project
  dataset_ids:
  - my-project:foo-dataset
  - my-project:bar-dataset
  bindings:
  - role: OWNER
    members:
    - group_email: my-project-owners@my-domain.com
  - role: WRITER
    members:
    - group_email: my-project-readwrite@my-domain.com
  - role: READER
    members:
    - group_email: my-project-readonly@my-domain.com
    - group_email: another-readonly-group@googlegroups.com
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conf, _ := getTestConfigAndProject(t, tc.configData)
			got, err := BigqueryRules(conf)
			if err != nil {
				t.Fatalf("BigqueryRules = %v", err)
			}

			wantYAML := globalBigqueryRule + tc.wantYAML + auditDatasetRule
			want := make([]BigqueryRule, 0)
			if err := yaml.Unmarshal([]byte(wantYAML), &want); err != nil {
				t.Fatalf("yaml.Unmarshal = %v", err)
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("rules differ (-got, +want):\n%v", diff)
			}
		})
	}
}
