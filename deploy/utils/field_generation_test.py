"""Tests for deploy.utils.field_generation."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest
import ruamel.yaml
from deploy.utils import field_generation
from deploy.utils import utils

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

  def test_is_generated_fields_exist(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    testcases = {
        'audit-project': True,
        'forseti-project': True,
        'data-project-01': True,
        'data-project-02': False,
    }
    for project_id, exist in testcases.items():
      self.assertEqual(
          field_generation.is_generated_fields_exist(project_id, overall_root),
          exist)

  def test_get_generated_fields_ref(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    project1_gf = field_generation.get_generated_fields_ref(
        'data-project-01', overall_root, False)
    project1_gf_modified = field_generation.get_generated_fields_ref(
        'data-project-01', overall_root, False)
    self.assertIs(project1_gf_modified, project1_gf)

  def test_get_generated_fields_copy(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    project1_gf = field_generation.get_generated_fields_copy(
        'data-project-01', overall_root)
    project1_gf_modified = field_generation.get_generated_fields_copy(
        'data-project-01', overall_root)
    self.assertIsNot(project1_gf_modified, project1_gf)

  def test_get_generated_fields_ref_exist(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    project1_gf = field_generation.get_generated_fields_ref(
        'data-project-01', overall_root)
    project1_gf_modified = field_generation.get_generated_fields_ref(
        'data-project-01', overall_root)
    self.assertIs(project1_gf_modified, project1_gf)

  def test_get_generated_fields_ref_not_exist(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    project2_gf = field_generation.get_generated_fields_ref(
        'data-project-02', overall_root)
    self.assertFalse(project2_gf)
    project2_gf_modified = field_generation.get_generated_fields_ref(
        'data-project-02', overall_root)
    self.assertIs(project2_gf_modified, project2_gf)

  def test_get_ref_not_exist_nor_create(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    with self.assertRaises(utils.InvalidConfigError):
      field_generation.get_generated_fields_ref('data-project-02', overall_root,
                                                False)

  def test_is_deployed(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    testcases = {
        'data-project-01': True,
        'data-project-02': False,
        'data-project-03': False,
    }
    for project_id, is_deployed in testcases.items():
      self.assertEqual(
          field_generation.is_deployed(project_id, overall_root), is_deployed)

  def test_forseti_service_generated_fields(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    self.assertEqual(
        field_generation.get_forseti_service_generated_fields(overall_root), {
            'service_account':
                'forseti-server-gcp-6fcf0fc@forseti-project.iam.gserviceaccount.com',
            'server_bucket':
                'gs://forseti-server-6fcf0fc/'
        })

  def test_convert_old_generated_fields_to_new(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_OLD_YAML_CONTENT)
    field_generation.convert_old_generated_fields_to_new(overall_root)
    overall_root_except = yaml.load(TEST_YAML_CONTENT)
    for project in overall_root_except[field_generation.GENERATED_FIELDS_NAME][
        field_generation._PROJECTS_TAG]:
      self.assertEqual(
          field_generation.get_generated_fields_copy(project,
                                                     overall_root_except),
          field_generation.get_generated_fields_copy(project, overall_root))
    self.assertEqual(
        field_generation.get_forseti_service_generated_fields(
            overall_root_except),
        field_generation.get_forseti_service_generated_fields(overall_root))

  def test_is_old_generated_fields_format_exist(self):
    yaml = ruamel.yaml.YAML()
    overall_root = yaml.load(TEST_YAML_CONTENT)
    overall_root.pop('generated_fields')
    self.assertFalse(
        field_generation.is_old_generated_fields_format_exist(overall_root))

    testcases = [
        overall_root['audit_logs_project'],
        overall_root['forseti']['project'],
        overall_root['forseti'],
        overall_root['projects'][1],
    ]

    for tc in testcases:
      tc['generated_fields'] = {}
      self.assertTrue(
          field_generation.is_old_generated_fields_format_exist(overall_root))
      tc.pop('generated_fields')


if __name__ == '__main__':
  absltest.main()
