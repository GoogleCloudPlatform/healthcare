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
"""Helper class for reading YAML files from GCS."""
import logging
from google.cloud import exceptions
from google.cloud import storage


class GcsClient(object):
  """Class for reading YAML files from GCS."""

  def __init__(self, config_bucket, storage_client=None):
    """Initialize.

    Args:
      config_bucket (str): Name of the GCS bucket that this client accesses.
      storage_client (storage.client.Client): Client to use for GCS requests. If
        not provided, a new Client will be created.
    """
    self._gcs = storage_client or storage.Client()
    self._config_bucket = config_bucket

  def get_blob(self, path):
    """Returns a blob for the specified path, or None if there is an error.

    Args:
      path (str): Path to the blob within GCS config bucket.

    Returns:
      storage.blob.Blob for the specified path, or None.
    """
    try:
      bucket = self._gcs.get_bucket(self._config_bucket)
      return bucket.get_blob(path)
    except exceptions.GoogleCloudError as e:
      # TODO: handle retriable errors here.
      logging.error('Error reading blob %s', e)
      return None

  def get_blobs(self, prefix=None, suffix=None):
    """Yields all blobs in the config bucket.

    An optional prefix and/or suffix can be provided.

    Args:
      prefix: If provided, prefix/path that all files must have.
      suffix: If provided, suffix that all files must have.

    Yields:
      Iterator[Blob]: An iterator of matching blobs.
    """
    bucket = self._gcs.get_bucket(self._config_bucket)
    try:
      for blob in bucket.list_blobs(prefix=prefix):
        if not suffix or blob.name.endswith(suffix):
          yield blob
    except exceptions.GoogleCloudError as e:
      logging.error('Error listing blobs %s', e)
      return
