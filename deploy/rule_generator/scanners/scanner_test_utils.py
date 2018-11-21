"""Test utils for rule generator scanners."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import yaml

from deploy.rule_generator import project_config

_OVERALL_DICT = {
    'domain': 'domain.com',
    'organization_id': '246801357924',
    'billing_account': '012345-6789AB-CDEF01',
    'allowed_apis': ['compute.googleapis.com', 'storage.googleapis.com'],
    'generated_fields': {
        'forseti_gcp_reader': 'forseti@sample-forseti.iam.gserviceaccount.com',
    },
}

_PROJECT_YAML = """
projects:
- project_id: %(project_id)s
  owners_group: %(project_id)s-owners@domain.com
  auditors_group: %(project_id)s-auditors@domain.com
  data_readwrite_groups:
  - %(project_id)s-readwrite@domain.com
  data_readonly_groups:
  - %(project_id)s-readonly@domain.com
  - %(project_id)s-readonly-external@domain.com
  generated_fields:
    project_number: %(project_num)s
    log_sink_service_account: audit-logs-bq@logging-%(project_num)s.iam.gserviceaccount.com
  data_buckets:
    - name_suffix: '-bucket'
      location: US-CENTRAL1
      storage_class: REGIONAL
  audit_logs:
    logs_gcs_bucket:
      location: US
      storage_class: MULTI_REGIONAL
      ttl_days: 365
    logs_bigquery_dataset:
      location: US
"""


def create_test_global_config():
  """Creates a global config dictionary for tests."""
  return _OVERALL_DICT


def create_test_project(project_id, project_num, extra_fields=None,
                        audit_logs_project=None):
  """Create a test project with optional extra project_config fields."""
  config_dict = yaml.load(_PROJECT_YAML % {
      'project_id': project_id, 'project_num': project_num})
  project = config_dict['projects'][0]
  if extra_fields:
    project.update(extra_fields)
  return project_config.ProjectConfig(overall=_OVERALL_DICT, project=project,
                                      audit_logs_project=audit_logs_project)


def create_test_projects(num_projects):
  """Create a list of test projects."""
  return [
      create_test_project(project_id='project-%s' % i, project_num=1000000 + i)
      for i in range(num_projects)
  ]
