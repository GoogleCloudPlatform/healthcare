"""Tests for rule_generator.scanners.log_sink_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import lien_scanner_rules
from deploy.rule_generator.scanners import scanner_test_utils

_EXPECTED_RULES_YAML = """
rules:
  - name: 'Require project deletion liens for all projects.'
    mode: required
    resource:
      - type: organization
        resource_ids:
          - '246801357924'
    restrictions: ['resourcemanager.projects.delete']
"""


class LienScannerRulesTest(absltest.TestCase):

  def test_generate_rules(self):
    got_rules = lien_scanner_rules.LienScannerRules().generate_rules(
        [], scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(_EXPECTED_RULES_YAML)
    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
