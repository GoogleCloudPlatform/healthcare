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
"""Tests for deploy.audit_backend.processors.gcs_client."""

from absl.testing import absltest

import mock
from google.cloud import exceptions
from google.cloud import storage

from deploy.audit_backend.processors import gcs_client

_TEST_FILE_NAMES = [
    'scanners/a_non_rule_file.yaml',
    'scanners/enabled_apis_rules.yaml',
    'scanners/iam_rules.yaml',
    'scanners/non_yaml_rules.txt',
    'scanners/retention_rules.yaml',
    'scanners/some_other_file.yaml',
]

_CONFIG_BUCKET = 'my-config-bucket'


class GcsClientTest(absltest.TestCase):

  def setUp(self):
    super(GcsClientTest, self).setUp()
    self.mock_blobs = []
    for name in _TEST_FILE_NAMES:
      blob = mock.create_autospec(storage.blob.Blob, spec_set=True)
      # Work-around to mock out an attribute called `name`.
      p = mock.PropertyMock(return_value=name)
      type(blob).name = p
      self.mock_blobs.append(blob)

    self.mock_bucket = mock.create_autospec(
        storage.bucket.Bucket, spec_set=True)
    self.mock_bucket.get_blob.return_value = self.mock_blobs[0]
    self.mock_bucket.list_blobs.return_value = self.mock_blobs

    self.mock_storage_client = mock.create_autospec(
        storage.client.Client, spec_set=True)
    self.mock_storage_client.get_bucket.return_value = self.mock_bucket

    self.gcs_client = gcs_client.GcsClient(
        _CONFIG_BUCKET, storage_client=self.mock_storage_client)

  def test_get_blob_succeeds(self):
    path = _TEST_FILE_NAMES[0]
    blob = self.gcs_client.get_blob(path)

    self.assertEqual(path, blob.name)

    # Check call was made for the right bucket and blob.
    self.mock_storage_client.get_bucket.assert_called_once_with(_CONFIG_BUCKET)
    self.mock_bucket.get_blob.assert_called_once_with(path)

  def test_get_blob_not_found_returns_none(self):
    self.mock_bucket.get_blob.return_value = None
    self.assertIsNone(self.gcs_client.get_blob('some_file'))

  def test_get_blob_error_returns_none(self):
    self.mock_bucket.get_blob.side_effect = exceptions.Forbidden(
        'Permission denied.')
    self.assertIsNone(self.gcs_client.get_blob('some_file'))

  def test_get_blobs_succeeds(self):
    blobs = self.gcs_client.get_blobs()
    actual_names = [blob.name for blob in blobs]

    self.assertEqual(_TEST_FILE_NAMES, actual_names)

    # Check call was made for the right bucket and blobs without prefix.
    self.mock_storage_client.get_bucket.assert_called_once_with(_CONFIG_BUCKET)
    self.mock_bucket.list_blobs.assert_called_once_with(prefix=None)

  def test_get_blobs_with_prefix_and_suffix(self):
    prefix = 'scanners/'
    suffix = '_rules.yaml'
    expected_names = [
        'scanners/enabled_apis_rules.yaml',
        'scanners/iam_rules.yaml',
        'scanners/retention_rules.yaml',
    ]

    blobs = self.gcs_client.get_blobs(prefix=prefix, suffix=suffix)
    actual_names = [blob.name for blob in blobs]

    self.assertEqual(expected_names, actual_names)

    # Check call was made for the right bucket and blobs with prefix.
    self.mock_storage_client.get_bucket.assert_called_once_with(_CONFIG_BUCKET)
    self.mock_bucket.list_blobs.assert_called_once_with(prefix=prefix)

  def test_get_blobs_empty_results_after_filtering(self):
    # No files returned have this suffix
    self.assertEqual([], list(self.gcs_client.get_blobs(suffix='.xml')))

  def test_get_blobs_empty_bucket(self):
    # No files returned from call to list_blobs.
    self.mock_bucket.list_blobs.return_value = []
    self.assertEqual([], list(self.gcs_client.get_blobs(prefix='nothing/')))

  def test_get_blobs_server_error(self):
    self.mock_bucket.list_blobs.side_effect = exceptions.ServerError(
        'Something went wrong.')
    self.assertEqual([], list(self.gcs_client.get_blobs()))


if __name__ == '__main__':
  absltest.main()
