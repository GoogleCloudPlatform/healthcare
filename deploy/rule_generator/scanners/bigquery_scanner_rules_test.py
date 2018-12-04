"""Tests for rule_generator.scanners.bigquery_scanner_rules."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest

import yaml

from deploy.rule_generator.scanners import bigquery_scanner_rules as bq_rules
from deploy.rule_generator.scanners import scanner_test_utils

EXPECTED_GLOBAL_RULES_YAML = """
  - name: No public, domain or special group dataset access.
    mode: blacklist
    resource:
      - type: {global_resource_type}
        resource_ids: {global_resource_ids}
    dataset_ids:
      - '*'
    bindings:
      - role: '*'
        members:
          - domain: '*'
          - special_group: '*'
"""

TEST_DATASETS = {
    'bigquery_datasets': [
        {
            'name': 'us_data',
            'location': 'US'
        },
        {
            'name': 'euro_data',
            'location': 'EU'
        },
    ],
}

EXPECTED_PROJECT_RULES_YAML = """
  - name: 'Whitelist for dataset(s): project_1:us_data, project_1:euro_data'
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - project_1
    dataset_ids:
      - project_1:us_data
      - project_1:euro_data
    bindings:
      - role: OWNER
        members:
          - group_email: 'project_1-owners@domain.com'
      - role: WRITER
        members:
          - group_email: 'project_1-readwrite@domain.com'
      - role: READER
        members:
          - group_email: 'project_1-readonly@domain.com'
          - group_email: 'project_1-readonly-external@domain.com'
"""

EXPECTED_LOCAL_AUDIT_PROJECT_YAML = """
  - name: Whitelist for project project_1 audit logs
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - project_1
    dataset_ids:
      - project_1:audit_logs
    bindings:
      - role: OWNER
        members:
          - group_email: 'project_1-owners@domain.com'
      - role: WRITER
        members:
          - user_email: 'audit-logs-bq@logging-123456.iam.gserviceaccount.com'
      - role: READER
        members:
          - group_email: 'project_1-auditors@domain.com'
"""


class BigQueryScannerRulesTest(absltest.TestCase):

  def test_generate_rules_no_projects(self):
    projects = []
    got_rules = bq_rules.BigQueryScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())
    want_rules = yaml.load('rules:\n{}'.format(
        EXPECTED_GLOBAL_RULES_YAML.format(
            global_resource_type='organization',
            global_resource_ids=['246801357924'])))
    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_project_with_local_audit_logs(self):
    projects = [scanner_test_utils.create_test_project(
        project_id='project_1', project_num=123456, extra_fields=TEST_DATASETS)]
    got_rules = bq_rules.BigQueryScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())

    want_rules = yaml.load('rules:\n{}\n{}\n{}'.format(
        EXPECTED_GLOBAL_RULES_YAML.format(
            global_resource_type='organization',
            global_resource_ids=['246801357924']), EXPECTED_PROJECT_RULES_YAML,
        EXPECTED_LOCAL_AUDIT_PROJECT_YAML))

    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_project_with_remote_audit_logs(self):
    expected_audit_project_yaml = """
  - name: Whitelist for project project_1 audit logs
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - project_1-audit
    dataset_ids:
      - project_1-audit:audit_logs
    bindings:
      - role: OWNER
        members:
          - group_email: 'project_1-audit_owners@domain.com'
      - role: WRITER
        members:
          - user_email: 'audit-logs-bq@logging-123456.iam.gserviceaccount.com'
      - role: READER
        members:
          - group_email: 'project_1-auditors@domain.com'
    """

    extra_fields = {
        'audit_logs': {
            'logs_bigquery_dataset': {
                'name': 'audit_logs',
                'location': 'US',
            },
        }
    }
    extra_fields.update(TEST_DATASETS)

    projects = [
        scanner_test_utils.create_test_project(
            project_id='project_1',
            project_num=123456,
            extra_fields=extra_fields,
            audit_logs_project={
                'project_id': 'project_1-audit',
                'owners_group': 'project_1-audit_owners@domain.com',
            })
    ]
    got_rules = bq_rules.BigQueryScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())

    want_rules = yaml.load('rules:\n{}\n{}\n{}'.format(
        EXPECTED_GLOBAL_RULES_YAML.format(
            global_resource_type='organization',
            global_resource_ids=['246801357924']), EXPECTED_PROJECT_RULES_YAML,
        expected_audit_project_yaml))

    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_project_with_additional_permissions(self):
    extra_permissions_rule_yaml = """
  - name: 'Whitelist for dataset(s): project_1:extra_data'
    mode: whitelist
    resource:
      - type: project
        resource_ids:
          - project_1
    dataset_ids:
      - project_1:extra_data
    bindings:
      - role: OWNER
        members:
          - group_email: 'project_1-owners@domain.com'
          - group_email: 'an_extra_group@googlegroups.com'
      - role: WRITER
        members:
          - group_email: 'project_1-readwrite@domain.com'
          - user_email: 'generator@project1.serviceacct.com'
          - user_email: 'extra-user@gmail.com'
      - role: READER
        members:
          - group_email: 'project_1-readonly@domain.com'
          - group_email: 'project_1-readonly-external@domain.com'
    """
    datasets = {
        'bigquery_datasets': [
            {
                'name': 'us_data',
                'location': 'US'
            },
            {
                'name': 'extra_data',
                'location': 'US',
                'additional_dataset_permissions': {
                    'owners': ['group:an_extra_group@googlegroups.com'],
                    'readwrite': [
                        'serviceAccount:generator@project1.serviceacct.com',
                        'user:extra-user@gmail.com'],
                }
            },
            {
                'name': 'euro_data',
                'location': 'EU'
            },
        ]
    }

    projects = [scanner_test_utils.create_test_project(
        project_id='project_1', project_num=123456, extra_fields=datasets)]

    got_rules = bq_rules.BigQueryScannerRules().generate_rules(
        projects, scanner_test_utils.create_test_global_config())

    want_rules = yaml.load('rules:\n{}\n{}\n{}\n{}'.format(
        EXPECTED_GLOBAL_RULES_YAML.format(
            global_resource_type='organization',
            global_resource_ids=['246801357924']), EXPECTED_PROJECT_RULES_YAML,
        extra_permissions_rule_yaml, EXPECTED_LOCAL_AUDIT_PROJECT_YAML))

    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_no_org_id(self):
    global_config = scanner_test_utils.create_test_global_config()
    global_config.pop('organization_id')
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project_1',
            project_num=123456,
            extra_fields=TEST_DATASETS)
    ]
    got_rules = bq_rules.BigQueryScannerRules().generate_rules(
        projects, global_config)
    want_rules = yaml.load('rules:\n{}\n{}\n{}'.format(
        EXPECTED_GLOBAL_RULES_YAML.format(
            global_resource_type='folder',
            global_resource_ids=['357801357924']), EXPECTED_PROJECT_RULES_YAML,
        EXPECTED_LOCAL_AUDIT_PROJECT_YAML))

    self.assertEqual(got_rules, want_rules)

  def test_generate_rules_no_org_and_folder_id(self):
    global_config = scanner_test_utils.create_test_global_config()
    global_config.pop('organization_id')
    global_config.pop('folder_id')
    projects = [
        scanner_test_utils.create_test_project(
            project_id='project_1',
            project_num=123456,
            extra_fields=TEST_DATASETS)
    ]
    got_rules = bq_rules.BigQueryScannerRules().generate_rules(
        projects, global_config)
    want_rules = yaml.load('rules:\n{}\n{}\n{}'.format(
        EXPECTED_GLOBAL_RULES_YAML.format(
            global_resource_type='project', global_resource_ids=['project_1']),
        EXPECTED_PROJECT_RULES_YAML, EXPECTED_LOCAL_AUDIT_PROJECT_YAML))

    self.assertEqual(got_rules, want_rules)


if __name__ == '__main__':
  absltest.main()
