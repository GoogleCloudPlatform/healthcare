# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Tests for deploy.audit_backend.processors.project_config_processor."""

from absl.testing import absltest

import mock
import yaml
from google.cloud import exceptions
from google.cloud import storage

from deploy.audit_backend.processors import audit_api_client
from deploy.audit_backend.processors import gcs_client
from deploy.audit_backend.processors import project_config_processor

_PROJECT_CONFIG_YAML = """
overall:
  organization_id: '246801357924'
  billing_account: 012345-6789AB-CDEF01

projects:
- project_id: project_1
- project_id: project_2
"""


class ProjectConfigProcessorTest(absltest.TestCase):

  def setUp(self):
    super(ProjectConfigProcessorTest, self).setUp()
    self.mock_blob = mock.create_autospec(
        storage.blob.Blob, spec_set=True, md5_hash='new_hash')
    self.mock_gcs_client = mock.create_autospec(
        gcs_client.GcsClient, spec_set=True)
    self.mock_gcs_client.get_blob.return_value = self.mock_blob
    self.mock_api_client = mock.create_autospec(
        audit_api_client.AuditApiClient, spec_set=True)
    # API has an older version of the project config file.
    self.mock_api_client.get_latest_config_md5_hash.return_value = 'old_hash'
    self.config_path = 'some/project/config.yaml'

  def test_config_is_updated(self):
    """Tests a new config is written when the config file changes hash."""
    self.mock_blob.download_as_string.return_value = _PROJECT_CONFIG_YAML

    self.assertTrue(
        project_config_processor.update_project_config(self.config_path,
                                                       self.mock_gcs_client,
                                                       self.mock_api_client))

    # New config is saved to API.
    expected_dict = yaml.safe_load(_PROJECT_CONFIG_YAML)
    self.mock_api_client.get_latest_config_md5_hash.assert_called_once()
    self.mock_api_client.create_new_project_config.assert_called_once_with(
        expected_dict, 'new_hash')

  def test_config_is_already_up_to_date(self):
    """Tests no config is written if the config file hasn't changed hash."""
    # API has the same version of the project config file.
    self.mock_api_client.get_latest_config_md5_hash.return_value = 'new_hash'

    self.assertFalse(
        project_config_processor.update_project_config(self.config_path,
                                                       self.mock_gcs_client,
                                                       self.mock_api_client))

    # Config file is not downloaded, and no new config is saved to API.
    self.mock_blob.download_as_string.assert_not_called()
    self.mock_api_client.create_new_project_config.assert_not_called()

  def test_config_file_not_found(self):
    """Tests handling of a missing config YAML file."""
    # Config file is not found.
    self.mock_gcs_client.get_blob.return_value = None

    self.assertFalse(
        project_config_processor.update_project_config(self.config_path,
                                                       self.mock_gcs_client,
                                                       self.mock_api_client))

    # No new config is saved to API.
    self.mock_api_client.create_new_project_config.assert_not_called()

  def test_error_reading_audit_api(self):
    """Tests handling of an error while reading the saved ProjectConfig."""
    # Audit API throws an exception while reading the latest ProjectConfig.
    self.mock_api_client.get_latest_config_md5_hash.side_effect = (
        exceptions.Forbidden('Permission denied.'))

    self.assertFalse(
        project_config_processor.update_project_config(self.config_path,
                                                       self.mock_gcs_client,
                                                       self.mock_api_client))

    # Config file is not downloaded, and no new config is saved to API.
    self.mock_blob.download_as_string.assert_not_called()
    self.mock_api_client.create_new_project_config.assert_not_called()

  def test_error_downloding_config_file(self):
    """Tests handling of an error while downloading the config YAML file."""
    # Storage API throws an exception while downloading the config file.
    self.mock_blob.download_as_string.side_effect = exceptions.ServerError(
        'Server Error')

    self.assertFalse(
        project_config_processor.update_project_config(self.config_path,
                                                       self.mock_gcs_client,
                                                       self.mock_api_client))

    # No new config is saved to the API.
    self.mock_api_client.create_new_project_config.assert_not_called()

  def test_bad_config_file(self):
    """Tests a config file that cannot be parsed."""
    self.mock_blob.download_as_string.return_value = 'a: "bad YAML'

    self.assertFalse(
        project_config_processor.update_project_config(self.config_path,
                                                       self.mock_gcs_client,
                                                       self.mock_api_client))

    # No new config is saved to API.
    self.mock_api_client.create_new_project_config.assert_not_called()

  def test_empty_config_file(self):
    """Tests a config file that is empty."""
    self.mock_blob.download_as_string.return_value = ''

    self.assertFalse(
        project_config_processor.update_project_config(self.config_path,
                                                       self.mock_gcs_client,
                                                       self.mock_api_client))

    # No new config is saved to API.
    self.mock_api_client.create_new_project_config.assert_not_called()

  def test_error_saving_config_file(self):
    """Tests a failed call to save the config to the Audit API."""
    self.mock_blob.download_as_string.return_value = _PROJECT_CONFIG_YAML
    self.mock_api_client.create_new_project_config.side_effect = (
        exceptions.ServerError('Server Error'))

    self.assertFalse(
        project_config_processor.update_project_config(self.config_path,
                                                       self.mock_gcs_client,
                                                       self.mock_api_client))


if __name__ == '__main__':
  absltest.main()
