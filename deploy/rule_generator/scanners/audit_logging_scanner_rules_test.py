"""Tests for rule_generator.scanners.audit_logging_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import audit_logging_scanner_rules as alsr
from deploy.rule_generator.scanners import scanner_test_utils

_EXPECTED_RULES_YAML = """
rules:
  - name: Require all Cloud Audit logs.
    resource:
      - type: project
        resource_ids:
          - '*'
    service: allServices
    log_types:
      - 'ADMIN_READ'
      - 'DATA_READ'
      - 'DATA_WRITE'
"""


class AuditLoggingScannerRulesTest(absltest.TestCase):

  def test_generate_rules(self):
    got_rules = alsr.AuditLoggingScannerRules().generate_rules(
        scanner_test_utils.create_test_projects(5),
        scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML)
    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
