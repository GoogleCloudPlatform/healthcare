package rulegen

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

// TODO: add test for remote audit project
var iamRulesConfigData = &ConfigData{`
resources:
- gcs_bucket:
    properties:
      name: foo-bucket
      location: us-east1
      bindings:
      - role: roles/storage.objectViewer
        members:
        - group:internal-bucket-viewers@my-domain.com
        - group:external-bucket-viewers@custom.com
- gcs_bucket:
    properties:
      name: bar-bucket
      location: us-east1
      bindings:
      - role: roles/storage.objectViewer
        members:
        - group:internal-bucket-viewers@my-domain.com
        - group:external-bucket-viewers@custom.com
- iam_policy:
    name: foo-policy
    properties:
      roles:
      - role: roles/viewer
        members:
        - group:internal-project-viewers@my-domain.com
        - group:external-project-viewers@custom.com
`}

const wantIAMRulesYAML = `
- name: All projects must have an owner group from the domain.
  mode: required
  resource:
  - type: project
    applies_to: self
    resource_ids:
    - '*'
  inherit_from_parents: true
  bindings:
  - role: roles/owner
    members:
    - group:*@my-domain.com
- name: All billing account roles must be groups from the domain.
  mode: whitelist
  resource:
  - type: billing_account
    applies_to: self
    resource_ids:
    - '*'
  inherit_from_parents: false
  bindings:
  - role: '*'
    members:
    - group:*@my-domain.com
- name: Global whitelist of allowed members for project roles.
  mode: whitelist
  resource:
  - type: project
    applies_to: self
    resource_ids:
    - '*'
  inherit_from_parents: true
  bindings:
  - role: '*'
    members:
    - 'group:*@my-domain.com'
    - 'serviceAccount:*.gserviceaccount.com'
    - 'group:external-project-viewers@custom.com'
- name: Global whitelist of allowed members for bucket roles.
  mode: whitelist
  resource:
  - type: bucket
    applies_to: self
    resource_ids:
    - '*'
  inherit_from_parents: true
  bindings:
  - role: '*'
    members:
    - group:*@my-domain.com
    - user:*@my-domain.com
    - serviceAccount:*.gserviceaccount.com
    - user:nobody
    - group:cloud-storage-analytics@google.com
    - group:another-readonly-group@googlegroups.com
    - group:external-bucket-viewers@custom.com
- name: Role whitelist for project my-forseti-project log bucket(s).
  mode: whitelist
  resource:
  - type: bucket
    applies_to: self
    resource_ids:
    - my-forseti-project-logs
  inherit_from_parents: true
  bindings:
  - role: roles/owner
    members:
    - group:my-forseti-project-owners@my-domain.com
  - role: roles/storage.objectAdmin
    members:
    - user:nobody
  - role: roles/storage.objectCreator
    members:
    - group:cloud-storage-analytics@google.com
  - role: roles/storage.objectViewer
    members:
    - group:my-forseti-project-auditors@my-domain.com
- name: Role whitelist for project my-forseti-project.
  mode: whitelist
  resource:
  - type: project
    applies_to: self
    resource_ids:
    - my-forseti-project
  inherit_from_parents: true
  bindings:
  - role: roles/owner
    members:
    - group:my-forseti-project-owners@my-domain.com
  - role: roles/iam.securityReviewer
    members:
    - group:my-forseti-project-auditors@my-domain.com
    - serviceAccount:forseti@my-forseti-project.iam.gserviceaccount.com
  - role: roles/editor
    members:
    - serviceAccount:2222-compute@developer.gserviceaccount.com
    - serviceAccount:2222@cloudservices.gserviceaccount.com
    - serviceAccount:service-2222@containerregistry.iam.gserviceaccount.com
- name: Role whitelist for project my-project log bucket(s).
  mode: whitelist
  resource:
  - type: bucket
    applies_to: self
    resource_ids:
    - my-project-logs
  inherit_from_parents: true
  bindings:
  - role: roles/owner
    members:
    - group:my-project-owners@my-domain.com
  - role: roles/storage.objectAdmin
    members:
    - user:nobody
  - role: roles/storage.objectCreator
    members:
    - group:cloud-storage-analytics@google.com
  - role: roles/storage.objectViewer
    members:
    - group:my-project-auditors@my-domain.com
- name: Role whitelist for project my-project.
  mode: whitelist
  resource:
  - type: project
    applies_to: self
    resource_ids:
    - my-project
  inherit_from_parents: true
  bindings:
  - role: roles/owner
    members:
    - group:my-project-owners@my-domain.com
  - role: roles/iam.securityReviewer
    members:
    - group:my-project-auditors@my-domain.com
    - serviceAccount:forseti@my-forseti-project.iam.gserviceaccount.com
  - role: roles/editor
    members:
    - group:my-project-editors@my-domain.com
    - serviceAccount:1111-compute@developer.gserviceaccount.com
    - serviceAccount:1111@cloudservices.gserviceaccount.com
    - serviceAccount:service-1111@containerregistry.iam.gserviceaccount.com
  - role: roles/viewer
    members:
    - group:internal-project-viewers@my-domain.com
    - group:external-project-viewers@custom.com
- name: 'Role whitelist for project my-project bucket(s): foo-bucket, bar-bucket.'
  mode: whitelist
  resource:
  - type: bucket
    applies_to: self
    resource_ids:
    - foo-bucket
    - bar-bucket
  inherit_from_parents: true
  bindings:
  - role: roles/storage.admin
    members:
    - group:my-project-owners@my-domain.com
  - role: roles/storage.objectAdmin
    members:
    - group:my-project-readwrite@my-domain.com
  - role: roles/storage.objectViewer
    members:
    - group:my-project-readonly@my-domain.com
    - group:another-readonly-group@googlegroups.com
    - group:internal-bucket-viewers@my-domain.com
    - group:external-bucket-viewers@custom.com
`

func TestIAMRules(t *testing.T) {
	config, _ := getTestConfigAndProject(t, iamRulesConfigData)
	got, err := IAMRules(config)
	if err != nil {
		t.Fatalf("IAMRules = %v", err)
	}

	want := make([]IAMRule, 0)
	if err := yaml.Unmarshal([]byte(wantIAMRulesYAML), &want); err != nil {
		t.Fatalf("yaml.Unmarshal = %v", err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("rules differ (-got, +want):\n%v", diff)
	}
}
