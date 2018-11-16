"""Tests for rule_generator.scanners.iam_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import iam_scanner_rules as isr
from deploy.rule_generator.scanners  import scanner_test_utils

_EXPECTED_RULES_YAML = """
rules:
  - name: All projects must have an owner group from the domain
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
          - group:*@domain.com
  - name: Global whitelist of allowed members for project roles
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
          - 'group:*@domain.com'
          - 'serviceAccount:*.gserviceaccount.com'
          - 'group:project_1_queriers@custom.com'
  - name: Global whitelist of allowed members for bucket roles
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
          - group:*@domain.com
          - serviceAccount:*.gserviceaccount.com
          - user:*@domain.com
          - group:cloud-storage-analytics@google.com
          - group:project_2-bucket@custom.com
  - name: Role whitelist for project project_1
    mode: whitelist
    resource:
      - type: project
        applies_to: self
        resource_ids:
          - 'project_1'
    inherit_from_parents: true
    bindings:
      - role: 'roles/bigquery.dataViewer'
        members:
          - group:project_1_queriers@custom.com
      - role: 'roles/editors'
        members:
          - serviceAccount:123456-compute@developer.gserviceaccount.com
          - serviceAccount:123456@cloudservices.gserviceaccount.com
          - serviceAccount:service-123456@containerregistry.iam.gserviceaccount.com
      - role: 'roles/iam.securityReviewer'
        members:
          - group:project_1-auditors@domain.com
          - serviceAccount:forseti@sample-forseti.iam.gserviceaccount.com
      - role: 'roles/owner'
        members:
          - group:project_1-owners@domain.com
  - name: Role whitelist 1 for buckets in project_1
    mode: whitelist
    resource:
      - type: bucket
        applies_to: self
        resource_ids:
          - 'project_1-logs'
    inherit_from_parents: true
    bindings:
      - role: roles/storage.admin
        members:
          - group:project_1-owners@domain.com
      - role: roles/storage.objectAdmin
        members:
        - user:nobody
      - role: roles/storage.objectCreator
        members:
          - group:cloud-storage-analytics@google.com
      - role: roles/storage.objectViewer
        members:
          - group:project_1-auditors@domain.com
  - name: Role whitelist 2 for buckets in project_1
    mode: whitelist
    resource:
      - type: bucket
        applies_to: self
        resource_ids:
          - 'project_1-data'
          - 'project_1-more-data'
          - 'project_1-euro-data'
    inherit_from_parents: true
    bindings:
      - role: roles/storage.admin
        members:
          - group:project_1-owners@domain.com
      - role: roles/storage.objectAdmin
        members:
        - group:project_1-readwrite@domain.com
      - role: roles/storage.objectCreator
        members:
        - user:nobody
      - role: roles/storage.objectViewer
        members:
        - group:project_1-readonly@domain.com
        - group:project_1-readonly-external@domain.com
  - name: Role whitelist for project project_2
    mode: whitelist
    resource:
      - type: project
        applies_to: self
        resource_ids:
          - 'project_2'
    inherit_from_parents: true
    bindings:
      - role: 'roles/editors'
        members:
          - group:project_2_editors@domain.com
          - serviceAccount:789012-compute@developer.gserviceaccount.com
          - serviceAccount:789012@cloudservices.gserviceaccount.com
          - serviceAccount:service-789012@containerregistry.iam.gserviceaccount.com
      - role: 'roles/iam.securityReviewer'
        members:
          - group:project_2-auditors@domain.com
          - serviceAccount:forseti@sample-forseti.iam.gserviceaccount.com
      - role: 'roles/owner'
        members:
          - group:project_2-owners@domain.com
  - name: Role whitelist 1 for buckets in project_2
    mode: whitelist
    resource:
      - type: bucket
        applies_to: self
        resource_ids:
          - 'project_2-data'
    inherit_from_parents: true
    bindings:
      - role: roles/storage.admin
        members:
          - group:project_2-owners@domain.com
      - role: roles/storage.objectAdmin
        members:
        - group:project_2-readwrite@domain.com
      - role: roles/storage.objectCreator
        members:
        - user:nobody
      - role: roles/storage.objectViewer
        members:
        - group:project_2-readonly@domain.com
        - group:project_2-bucket@custom.com
"""


class IamScannerRulesTest(absltest.TestCase):

  def test_generate_rules(self):
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project_1', project_num=123456,
            extra_fields={
                'additional_project_permissions': [{
                    'roles': ['roles/bigquery.dataViewer'],
                    'members': ['group:project_1_queriers@custom.com'],
                }],
                'data_buckets': [
                    {'name_suffix': '-data'},
                    {'name_suffix': '-more-data'},
                    {'name_suffix': '-euro-data'},
                ]
            }
        ),
        scanner_test_utils.create_test_project(
            project_id='project_2', project_num=789012,
            extra_fields={
                'editors_groups': ['project_2_editors@domain.com'],
                'data_readwrite_groups': ['project_2-readwrite@domain.com'],
                'data_readonly_groups': [
                    'project_2-readonly@domain.com',
                    'project_2-bucket@custom.com',
                ],
                'data_buckets': [{'name_suffix': '-data'}],
                'audit_logs': {
                    'logs_bigquery_dataset': {
                        'name': 'project_2_logs',
                        'location': 'US',
                    },
                },
            },
            audit_logs_project={
                'project_id': 'audit-logs',
                'owners_group': 'audit-logs_owners@domain.com',
            }
        ),
    ]

    got_rules = isr.IamScannerRules.generate_rules(
        projects, scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML)
    self.assertDictEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
