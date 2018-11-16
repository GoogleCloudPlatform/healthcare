"""Tests for rule_generator.scanners.enabled_apis_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import enabled_apis_scanner_rules as easr
from deploy.rule_generator.scanners import scanner_test_utils

_EXPECTED_RULES_YAML = """
rules:
  - name: 'Global API whitelist.'
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - '*'
    services:
      - 'compute.googleapis.com'
      - 'storage.googleapis.com'
  - name: 'API whitelist for project_1'
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - 'project_1'
    services:
    - bigquery-json.googleapis.com
    - servicemanagement.googleapis.com
    - monitoring.googleapis.com
    - storage-api.googleapis.com
    - logging.googleapis.com
    - storage-component.googleapis.com
  - name: 'API whitelist for project_3'
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - 'project_3'
    services:
    - pubsub.googleapis.com
"""


class EnabledApisScannerRulesTest(absltest.TestCase):

  def test_generate_rules(self):
    # Project-specific whitelists are only generated for projects with a
    # non-empty list of enabled_apis.
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project_1', project_num=100000001,
            extra_fields={
                'enabled_apis': [
                    'bigquery-json.googleapis.com',
                    'servicemanagement.googleapis.com',
                    'monitoring.googleapis.com', 'storage-api.googleapis.com',
                    'logging.googleapis.com', 'storage-component.googleapis.com'
                ]
            }
        ),
        scanner_test_utils.create_test_project(
            project_id='project_2', project_num=100000002,
            extra_fields={'enabled_apis': []}
        ),
        scanner_test_utils.create_test_project(
            project_id='project_3', project_num=100000003,
            extra_fields={'enabled_apis': ['pubsub.googleapis.com']}
        ),
        scanner_test_utils.create_test_project(
            project_id='project_4', project_num=100000004,
        ),
    ]

    got_rules = easr.EnabledApisScannerRules().generate_rules(
        projects,
        scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML)
    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
