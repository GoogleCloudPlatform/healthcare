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
  - name: Project project-1 resource whitelist for location US.
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - project-1
    applies_to:
      - type: dataset
        resource_ids:
          - project-1:dataset
    locations:
      - US
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
  - name: Project project-1 audit logs bucket location whitelist.
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
  - name: Project project-1 audit logs dataset location whitelist.
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - {audit_logs_project_id}
    applies_to:
      - type: dataset
        resource_ids:
          - {audit_logs_dataset_id}
    locations:
      - US
"""


class LocationScannerRulesTest(absltest.TestCase):

  def test_generate_rules_local_audit_logs(self):
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project-1', project_num=123456,
            extra_fields={
                'bigquery_datasets': [{
                    'name': 'dataset',
                    'location': 'US',
                }],
            },
        ),
    ]
    got_rules = location_scanner_rules.LocationScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML.format(
        audit_logs_project_id='project-1',
        audit_logs_bucket_id='project-1-logs',
        audit_logs_dataset_id='project-1:audit_logs',
    ))
    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_local_remote_logs(self):
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project-1', project_num=123456,
            extra_fields={
                'bigquery_datasets': [{
                    'name': 'dataset',
                    'location': 'US',
                }],
                'gce_instances': [{
                    'name': 'instance',
                }],
                'audit_logs': {
                    'logs_gcs_bucket': {
                        'name': 'project-1-remote-logs',
                        'location': 'US',
                    },
                    'logs_bigquery_dataset': {
                        'name': 'project-1-remote-dataset',
                        'location': 'US',
                    },
                },
            },
            audit_logs_project={
                'project_id': 'project-1-audit',
                'owners_group': 'project-1-owners@google.com',
            }
        ),
    ]

    got_rules = location_scanner_rules.LocationScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML.format(
        audit_logs_project_id='project-1-audit',
        audit_logs_bucket_id='project-1-remote-logs',
        audit_logs_dataset_id='project-1-audit:project-1-remote-dataset',
    ))
    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
