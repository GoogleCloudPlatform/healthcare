"""Tests for rule_generator.scanners.cloudsql_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import cloudsql_scanner_rules as cssr
from deploy.rule_generator.scanners import scanner_test_utils

_EXPECTED_RULES_YAML = """
rules:
  - name: Disallow publicly exposed cloudsql instances (SSL disabled).
    instance_name: '*'
    authorized_networks: '0.0.0.0/0'
    ssl_enabled: 'False'
    resource:
      - type: organization
        resource_ids:
          - '246801357924'
  - name: Disallow publicly exposed cloudsql instances (SSL enabled).
    instance_name: '*'
    authorized_networks: '0.0.0.0/0'
    ssl_enabled: 'True'
    resource:
      - type: organization
        resource_ids:
          - '246801357924'
"""


class CloudSqlScannerRulesTest(absltest.TestCase):

  def test_generate_rules(self):
    got_rules = cssr.CloudSqlScannerRules().generate_rules(
        scanner_test_utils.create_test_projects(5),
        scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML)
    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
