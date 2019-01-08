"""Tests for rule_generator.scanners.resource_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import resource_scanner_rules
from deploy.rule_generator.scanners import scanner_test_utils

_EXPECTED_RULES_YAML = """
rules:
  - name: 'Project resource trees.'
    mode: required
    resource_types:
      - project
      - bucket
      - dataset
      - instance
    resource_trees:
      - type: project
        resource_id: '*'
      - type: project
        resource_id: project-1
        children:
          - type: bucket
            resource_id: project-1-bucket
          - type: dataset
            resource_id: project-1:dataset
          - type: instance
            resource_id: '123'
"""


class ResourceScannerRulesTest(absltest.TestCase):

  def test_generate_rules(self):
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project-1',
            project_num=123456,
            extra_fields={
                'bigquery_datasets': [{
                    'name': 'dataset',
                    'location': 'US',
                }],
            },
        ),
    ]
    got_rules = resource_scanner_rules.ResourceScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML)
    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
