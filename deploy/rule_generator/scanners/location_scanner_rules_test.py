"""Tests for rule_generator.scanners.log_sink_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import location_scanner_rules
from deploy.rule_generator.scanners import scanner_test_utils

_EXPECTED_RULES_YAML = """
rules:
  - name: Project project-1 resource whitelist for location US-CENTRAL1.
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - project-1
    applies_to:
      - type: bucket
        resource_ids:
          - project-1-bucket
    locations:
      - US-CENTRAL1
  - name: Project project-1 audit logs location whitelist.
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - {audit_logs_project_id}
    applies_to:
      - type: bucket
        resource_ids:
          - {audit_logs_bucket_id}
    locations:
      - US
"""


class LocationScannerRulesTest(absltest.TestCase):

  def test_generate_rules_local_audit_logs(self):
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project-1', project_num=123456,
        ),
    ]
    got_rules = location_scanner_rules.LocationScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML.format(
        audit_logs_project_id='project-1',
        audit_logs_bucket_id='project-1-logs'
    ))
    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_local_remote_logs(self):
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project-1', project_num=123456,
            extra_fields={
                'audit_logs': {
                    'logs_gcs_bucket': {
                        'name': 'project-1-remote-logs',
                        'location': 'US',
                    },
                    # TODO: update test to check for dataset
                    # locations once support has been added in Forseti.
                    'logs_bigquery_dataset': {
                        'name': 'project-1-remote-dataset',
                        'location': 'EU',
                    },
                },
            },
            audit_logs_project={
                'project_id': 'project-1-audit',
                'owners_group': 'project-1-owners@domain.com',
            }
        ),
    ]

    got_rules = location_scanner_rules.LocationScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML.format(
        audit_logs_project_id='project-1-audit',
        audit_logs_bucket_id='project-1-remote-logs'
    ))
    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
