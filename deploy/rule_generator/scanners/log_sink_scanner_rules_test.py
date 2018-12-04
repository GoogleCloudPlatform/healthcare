"""Tests for rule_generator.scanners.log_sink_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import log_sink_scanner_rules as lssr
from deploy.rule_generator.scanners import scanner_test_utils

_EXPECTED_RULES_YAML = """
rules:
  - name: 'Require a BigQuery Log sink in all projects.'
    mode: required
    resource:
      - type: {global_resource_type}
        applies_to: {global_resource_applies_to}
        resource_ids: {global_resource_ids}
    sink:
      destination: 'bigquery.googleapis.com/*'
      filter: '*'
      include_children: '*'
  - name: 'Only allow BigQuery Log sinks in all projects.'
    mode: whitelist
    resource:
      - type: {global_resource_type}
        applies_to: {global_resource_applies_to}
        resource_ids: {global_resource_ids}
    sink:
      destination: 'bigquery.googleapis.com/*'
      filter: '*'
      include_children: '*'
  - name: 'Require Log sink for project project-1.'
    mode: required
    resource:
      - type: project
        applies_to: self
        resource_ids:
          - project-1
    sink:
      destination: >-
        bigquery.googleapis.com/projects/project-1/datasets/audit_logs
      filter: '*'
      include_children: '*'
  - name: 'Whitelist Log sink for project project-1.'
    mode: whitelist
    resource:
      - type: project
        applies_to: self
        resource_ids:
          - project-1
    sink:
      destination: >-
        bigquery.googleapis.com/projects/project-1/datasets/audit_logs
      filter: '*'
      include_children: '*'
  - name: 'Require Log sink for project project-2.'
    mode: required
    resource:
      - type: project
        applies_to: self
        resource_ids:
          - project-2
    sink:
      destination: >-
        bigquery.googleapis.com/projects/audit-logs/datasets/project_2_logs
      filter: '*'
      include_children: '*'
  - name: 'Whitelist Log sink for project project-2.'
    mode: whitelist
    resource:
      - type: project
        applies_to: self
        resource_ids:
          - project-2
    sink:
      destination: >-
        bigquery.googleapis.com/projects/audit-logs/datasets/project_2_logs
      filter: '*'
      include_children: '*'
"""

_PROJECTS = [
    scanner_test_utils.create_test_project(
        project_id='project-1', project_num=123456),
    scanner_test_utils.create_test_project(
        project_id='project-2',
        project_num=789012,
        extra_fields={
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
        })
]


class LogSinkScannerRulesTest(absltest.TestCase):

  def test_generate_rules(self):
    got_rules = lssr.LogSinkScannerRules().generate_rules(
        _PROJECTS, scanner_test_utils.create_test_global_config())
    want_rules = yaml.load(
        _EXPECTED_RULES_YAML.format(
            global_resource_type='organization',
            global_resource_applies_to='children',
            global_resource_ids=['246801357924']))
    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_no_org_id(self):
    global_config = scanner_test_utils.create_test_global_config()
    global_config.pop('organization_id')
    got_rules = lssr.LogSinkScannerRules().generate_rules(
        _PROJECTS, global_config)
    want_rules = yaml.load(
        _EXPECTED_RULES_YAML.format(
            global_resource_type='folder',
            global_resource_applies_to='children',
            global_resource_ids=['357801357924'],
        ))
    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_no_org_and_folder_id(self):
    global_config = scanner_test_utils.create_test_global_config()
    global_config.pop('organization_id')
    global_config.pop('folder_id')
    got_rules = lssr.LogSinkScannerRules().generate_rules(
        _PROJECTS, global_config)
    want_rules = yaml.load(
        _EXPECTED_RULES_YAML.format(
            global_resource_type='project',
            global_resource_applies_to='self',
            global_resource_ids=['project-1', 'project-2'],
        ))
    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
