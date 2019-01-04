"""Tests for rule_generator.project_config."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import copy

from absl.testing import absltest

import yaml

from deploy.rule_generator.project_config import ProjectConfig

TEST_PROJECT_YAML = """
overall:
  organization_id: '246801357924'
  billing_account: 012345-6789AB-CDEF01

forseti:
  project:
    project_id: forseti-project
    owners_group: forseti-project-owners@domain.com
    auditors_group: forseti-project-auditors@domain.com
    data_readwrite_groups:
      - forseti-project-readwrite@domain.com
    data_readonly_groups:
      - forseti-project-readonly@domain.com
    generated_fields:
      project_number: 9999,
      log_sink_service_account: forseti-logs@logging-9999.iam.gserviceaccount.com
  generated_fields:
    service_account: forseti@sample-forseti.iam.gserviceaccount.com
    server_bucket: gs://forseti-project-server/

projects:
- project_id: sample-data
  owners_group: sample-data-owners@domain.com
  auditors_group: sample-data-auditors@domain.com
  data_readwrite_groups:
  - sample-data-readwrite@domain.com
  data_readonly_groups:
  - sample-data-readonly@domain.com
  - sample-data-external@domain.com
  additional_project_permissions:
  - roles:
    - roles/bigquery.dataViewer
    - roles/ml.developer
    members:
    - group:sample-data-readwrite@domain.com
    - group:sample-data-readonly@domain.com
    - group:sample-data-external@domain.com
  audit_logs:
    logs_gcs_bucket:
      location: US
      storage_class: MULTI_REGIONAL
      ttl_days: 365
    logs_bigquery_dataset:
      location: US
  data_buckets:
  - name_suffix: '-raw'
    location: US-CENTRAL1
    storage_class: REGIONAL
  - name_suffix: '-processed'
    location: US-CENTRAL1
    storage_class: REGIONAL
    additional_bucket_permissions:
      owners:
       - 'serviceAccount:samples@system.gserviceaccount.com'
  bigquery_datasets:
  - name: 'data'
    location: US
  - name: 'more_data'
    location: US
    additional_dataset_permissions:
      readwrite:
       - 'serviceAccount:samples@system.gserviceaccount.com'
  - name: 'euro_data'
    location: EU
  stackdriver_alert_email: sample-data-auditors@domain.com
  enabled_apis:
  - monitoring.googleapis.com
  - logging.googleapis.com
  generated_fields:
    project_number: 123546879123
    log_sink_service_account: audit-logs-bq@logging-123.iam.gserviceaccount.com
"""


class ProjectConfigTest(absltest.TestCase):

  def test_load_valid_config(self):
    yaml_dict = yaml.load(TEST_PROJECT_YAML)
    project = ProjectConfig(
        project=yaml_dict['projects'][0],
        audit_logs_project=None,
        forseti=yaml_dict['forseti'])
    self.assertIsNotNone(project)

    self.assertEqual('sample-data', project.project_id)
    self.assertEqual(['monitoring.googleapis.com', 'logging.googleapis.com'],
                     project.enabled_apis)

    expected_proj_bindings = {
        'roles/owner': ['group:sample-data-owners@domain.com'],
        'roles/editor': [
            'serviceAccount:123546879123-compute@developer.gserviceaccount.com',
            'serviceAccount:123546879123@cloudservices.gserviceaccount.com',
            ('serviceAccount:service-123546879123@'
             'containerregistry.iam.gserviceaccount.com'),
        ],
        'roles/iam.securityReviewer': [
            'group:sample-data-auditors@domain.com',
            'serviceAccount:forseti@sample-forseti.iam.gserviceaccount.com',
        ],
        'roles/bigquery.dataViewer': [
            'group:sample-data-readwrite@domain.com',
            'group:sample-data-readonly@domain.com',
            'group:sample-data-external@domain.com',
        ],
        'roles/ml.developer': [
            'group:sample-data-readwrite@domain.com',
            'group:sample-data-readonly@domain.com',
            'group:sample-data-external@domain.com',
        ],
    }
    self.assertDictEqual(expected_proj_bindings, project.get_project_bindings())

    expected_log_bindings = {
        'roles/storage.admin': ['group:sample-data-owners@domain.com'],
        'roles/storage.objectAdmin': [],
        'roles/storage.objectViewer': ['group:sample-data-auditors@domain.com'],
        'roles/storage.objectCreator': [
            'group:cloud-storage-analytics@google.com'
        ],
    }
    expected_raw_data_bindings = {
        'roles/storage.admin': ['group:sample-data-owners@domain.com',],
        'roles/storage.objectAdmin': [
            'group:sample-data-readwrite@domain.com',
        ],
        'roles/storage.objectCreator': [],
        'roles/storage.objectViewer': [
            'group:sample-data-readonly@domain.com',
            'group:sample-data-external@domain.com',
        ],
    }
    expected_processed_data_bindings = copy.deepcopy(expected_raw_data_bindings)
    expected_processed_data_bindings['roles/storage.admin'].append(
        'serviceAccount:samples@system.gserviceaccount.com')
    expected_bucket_bindings = [
        (['sample-data-logs'], expected_log_bindings),
        (['sample-data-processed'], expected_processed_data_bindings),
        (['sample-data-raw'], expected_raw_data_bindings),
    ]
    self.assertEqual(expected_bucket_bindings, project.get_bucket_bindings())

    self.assertEqual(
        'bigquery.googleapis.com/projects/sample-data/datasets/audit_logs',
        project.get_audit_log_sink_destination())

  def test_get_project_bigquery_bindings(self):
    yaml_dict = yaml.load(TEST_PROJECT_YAML)
    project = ProjectConfig(
        project=yaml_dict['projects'][0],
        audit_logs_project=None,
        forseti=yaml_dict['forseti'])

    got_bindings = project.get_project_bigquery_bindings()
    default_bindings = [
        {
            'role': 'OWNER',
            'members': [{'group_email': 'sample-data-owners@domain.com'}],
        },
        {
            'role': 'WRITER',
            'members': [{'group_email': 'sample-data-readwrite@domain.com'}],
        },
        {
            'role': 'READER',
            'members': [
                {'group_email': 'sample-data-readonly@domain.com'},
                {'group_email': 'sample-data-external@domain.com'},
            ],
        },
    ]
    # Dataset more_data has an additional writer account.
    custom_bindings = copy.deepcopy(default_bindings)
    custom_bindings[1]['members'].append(
        {'user_email': 'samples@system.gserviceaccount.com'})
    want_bindings = [
        (['sample-data:data', 'sample-data:euro_data'], default_bindings),
        (['sample-data:more_data'], custom_bindings)
    ]

    self.assertEqual(got_bindings, want_bindings)

  def test_get_audit_logs_bigquery_bindings_local(self):
    yaml_dict = yaml.load(TEST_PROJECT_YAML)
    project = ProjectConfig(
        project=yaml_dict['projects'][0],
        audit_logs_project=None,
        forseti=yaml_dict['forseti'])

    got_bindings = project.get_audit_logs_bigquery_bindings()
    want_bindings = [
        {
            'role': 'OWNER',
            'members': [{'group_email': 'sample-data-owners@domain.com'}],
        },
        {
            'role': 'WRITER',
            'members': [{
                'user_email':
                    'audit-logs-bq@logging-123.iam.gserviceaccount.com'
            }],
        },
        {
            'role': 'READER',
            'members': [{'group_email': 'sample-data-auditors@domain.com'}],
        },
    ]

    self.assertEqual(got_bindings, want_bindings)

  def test_get_audit_logs_bigquery_bindings_remote(self):
    yaml_dict = yaml.load(TEST_PROJECT_YAML)
    project_dict = yaml_dict['projects'][0]
    # Set remote audit logs instead of local audit logs.
    project_dict['audit_logs'] = {
        'logs_bigquery_dataset': {
            'name': 'some_data_logs'
        },
    }
    audit_logs_project = {
        'project_id': 'audit-logs',
        'owners_group': 'auditors-owners@domain.com',
    }
    forseti = yaml_dict['forseti']
    project = ProjectConfig(
        project=project_dict,
        audit_logs_project=audit_logs_project,
        forseti=forseti)

    got_bindings = project.get_audit_logs_bigquery_bindings()
    want_bindings = [
        {
            'role': 'OWNER',
            'members': [{'group_email': 'auditors-owners@domain.com'}],
        },
        {
            'role': 'WRITER',
            'members': [{
                'user_email':
                    'audit-logs-bq@logging-123.iam.gserviceaccount.com'
            }],
        },
        {
            'role': 'READER',
            'members': [{'group_email': 'sample-data-auditors@domain.com'}],
        },
    ]

    self.assertEqual(got_bindings, want_bindings)

  def test_get_audit_log_sink_destination(self):
    # Local audit logs.
    yaml_dict = yaml.load(TEST_PROJECT_YAML)
    project_dict = yaml_dict['projects'][0]
    forseti = yaml_dict['forseti']
    project = ProjectConfig(
        project=project_dict, audit_logs_project=None, forseti=forseti)
    self.assertEqual(
        'bigquery.googleapis.com/projects/sample-data/datasets/audit_logs',
        project.get_audit_log_sink_destination())

    # Remote audit logs.
    project_dict['audit_logs'] = {
        'logs_bigquery_dataset': {
            'name': 'some_data_logs'
        },
    }
    audit_logs_project = {
        'project_id': 'audit-logs',
        'owners_group': 'auditors-owners@domain.com',
    }
    project = ProjectConfig(
        project=project_dict,
        audit_logs_project=audit_logs_project,
        forseti=forseti)
    self.assertEqual(
        'bigquery.googleapis.com/projects/audit-logs/datasets/some_data_logs',
        project.get_audit_log_sink_destination())


if __name__ == '__main__':
  absltest.main()
