"""Test utils for rule generator scanners."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import yaml

from deploy.rule_generator import project_config

_OVERALL_DICT = {
    'domain': 'domain.com',
    'organization_id': '246801357924',
    'folder_id': '357801357924',
    'billing_account': '012345-6789AB-CDEF01',
    'allowed_apis': ['compute.googleapis.com', 'storage.googleapis.com'],
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
  data_buckets:
  - name_suffix: '-bucket'
    location: US-CENTRAL1
    storage_class: REGIONAL
  gce_instances:
  - name: 'instance'
    zone: 'us-central1-f'
    machine_type: 'n1-standard-1'
    existing_boot_image: 'projects/debian-cloud/global/images/family/debian-9'
    start_vm: true
  audit_logs:
    logs_gcs_bucket:
      location: US
      storage_class: MULTI_REGIONAL
      ttl_days: 365
    logs_bigquery_dataset:
      location: US
"""

_GENERATED_FIELDS = """
generated_fields:
  forseti:
    service_account: forseti@sample-forseti.iam.gserviceaccount.com
    server_bucket: gs://forseti-project-server/
  projects:
    %(project_id)s:
      project_number: %(project_num)s
      log_sink_service_account: audit-logs-bq@logging-%(project_num)s.iam.gserviceaccount.com
      gce_instance_info:
      - name: 'instance'
        id: '123'
    forseti-project:
      project_number: 9999
      log_sink_service_account: forseti-logs@logging-9999.iam.gserviceaccount.com
"""


def create_test_global_config():
  """Creates a global config dictionary for tests."""
  return _OVERALL_DICT.copy()


def create_test_project(project_id, project_num, extra_fields=None,
                        audit_logs_project=None):
  """Create a test project with optional extra project_config fields."""
  config_dict = yaml.load(_PROJECT_YAML % {
      'project_id': project_id, 'project_num': project_num})
  project = config_dict['projects'][0]
  generated_fields = yaml.load(_GENERATED_FIELDS % {
      'project_id': project_id,
      'project_num': project_num
  })
  if extra_fields:
    project.update(extra_fields)
  print(generated_fields)
  print('xxxxxxxxxxxxxxxxxxxxxx')
  return project_config.ProjectConfig(
      project=project,
      audit_logs_project=audit_logs_project,
      generated_fields=generated_fields['generated_fields'])


def create_test_projects(num_projects):
  """Create a list of test projects."""
  return [
      create_test_project(project_id='project-%s' % i, project_num=1000000 + i)
      for i in range(num_projects)
  ]
