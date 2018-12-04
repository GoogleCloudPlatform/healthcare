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
      - type: {global_resource_type}
        resource_ids: {global_resource_ids}
    restrictions: ['resourcemanager.projects.delete']
"""


class LienScannerRulesTest(absltest.TestCase):

  def test_generate_rules(self):
    got_rules = lien_scanner_rules.LienScannerRules().generate_rules(
        [], scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(
        _EXPECTED_RULES_YAML.format(
            global_resource_type='organization',
            global_resource_ids=['246801357924']))
    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_no_org_id(self):
    global_config = scanner_test_utils.create_test_global_config()
    global_config.pop('organization_id')
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project_1', project_num=123456)
    ]
    got_rules = lien_scanner_rules.LienScannerRules().generate_rules(
        projects, global_config)
    want_rules = yaml.load(
        _EXPECTED_RULES_YAML.format(
            global_resource_type='folder',
            global_resource_ids=['357801357924']))

    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_no_org_and_folder_id(self):
    global_config = scanner_test_utils.create_test_global_config()
    global_config.pop('organization_id')
    global_config.pop('folder_id')
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project_1', project_num=123456)
    ]
    got_rules = lien_scanner_rules.LienScannerRules().generate_rules(
        projects, global_config)
    want_rules = yaml.load(
        _EXPECTED_RULES_YAML.format(
            global_resource_type='project', global_resource_ids=['project_1']))

    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
