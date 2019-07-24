"""Tests for deploy.utils.field_generation."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import tempfile
from absl.testing import absltest
import ruamel.yaml
from deploy.utils import field_generation

TEST_YAML_CONTENT = """
overall:
  organization_id: '433637338589'
  folder_id: '396521612403'
  billing_account: 00F4CE-59D8D8-2298AC

audit_logs_project:
  project_id: audit-project

forseti:
  project:
    project_id: forseti-project

projects:
- project_id: data-project-01
- project_id: data-project-02
- project_id: data-project-03

generated_fields:
  forseti:
    service_account: forseti-server-gcp-6fcf0fc@forseti-project.iam.gserviceaccount.com
    server_bucket: gs://forseti-server-6fcf0fc/
  projects:
    audit-project:
        log_sink_service_account: p111111111111-999999@gcp-sa-logging.iam.gserviceaccount.com
        project_number: '111111111111'
    forseti-project:
        log_sink_service_account: p222222222222-999999@gcp-sa-logging.iam.gserviceaccount.com
        project_number: '222222222222'
    data-project-01:
        log_sink_service_account: p333333333333-999999@gcp-sa-logging.iam.gserviceaccount.com
        project_number: '333333333333'
    data-project-03:
        failed_step: 15
"""

TEST_OLD_YAML_CONTENT = """
overall:
  organization_id: '433637338589'
  folder_id: '396521612403'
  billing_account: 00F4CE-59D8D8-2298AC

audit_logs_project:
  project_id: audit-project
  generated_fields:
    log_sink_service_account: p111111111111-999999@gcp-sa-logging.iam.gserviceaccount.com
    project_number: '111111111111'

forseti:
  project:
    project_id: forseti-project
    generated_fields:
      log_sink_service_account: p222222222222-999999@gcp-sa-logging.iam.gserviceaccount.com
      project_number: '222222222222'
  generated_fields:
    service_account: forseti-server-gcp-6fcf0fc@forseti-project.iam.gserviceaccount.com
    server_bucket: gs://forseti-server-6fcf0fc/

projects:
- project_id: data-project-01
  generated_fields:
    log_sink_service_account: p333333333333-999999@gcp-sa-logging.iam.gserviceaccount.com
    project_number: '333333333333'
- project_id: data-project-02
- project_id: data-project-03
  generated_fields:
    failed_step: 15
"""


class FieldGeneratingTest(absltest.TestCase):

  def test_update_generated_fields_noempty(self):
    with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml') as f:
      f.write(TEST_YAML_CONTENT)
      f.flush()
      yaml = ruamel.yaml.YAML()
      overall_root = yaml.load(TEST_YAML_CONTENT)
      overall_root['generated_fields']['projects']['data-project-03'][
          'failed_step'] = 16
      new_root = field_generation.update_generated_fields(f.name, overall_root)
      self.assertEqual(overall_root, new_root)


if __name__ == '__main__':
  absltest.main()
