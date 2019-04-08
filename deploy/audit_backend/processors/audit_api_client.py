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
"""Client for interfacing with the Cloud HCLS Audit API.

The client wraps API calls made by the dashboard backend for all required
operations.

Currently, this client fakes API calls by reading and writing resources in
memory. Once the API is launched, this client will be replaced with one backed
by real API calls.
"""
import datetime
import logging

from . import audit_api_resources


class AuditApiClient(object):
  """Client for interfacing with the Cloud HCLS Audit API."""

  def __init__(self, now=datetime.datetime.utcnow):
    """Initializer.

    Args:
      now: Callable that returns the current time as a datetime.datetime.
    """
    self._now = now
    # Ordered list of pairs of (ProjectConfig, Dict[str -> Scanner]) resources.
    # Most recent ProjectConfig saved last.
    self._saved_configs = []

  def get_latest_config_md5_hash(self):
    """Returns the md5_hash for the latest Project Config saved in the API.

    Returns:
      A string with the MD5 hash of the latest ProjectConfig, or None if there
      is no saved config.

    Raises:
      exceptions.GoogleCloudError if there is an error calling the API.
    """
    logging.info('Getting latest ProjectConfig from API.')
    if not self._saved_configs:
      return None
    return self._saved_configs[-1][0].hash

  def create_new_project_config(self, config_dict, config_md5_hash):
    """Creates and saves a new project config with the given config and hash.

    Args:
      config_dict (dict): a dictionary containing the parsed contents of the
        project config YAML file.
      config_md5_hash (str): the MD5 hash of the project config YAML file.

    Raises:
      exceptions.GoogleCloudError if there is an error calling the API.
    """
    now = self._now()
    project_config = audit_api_resources.ProjectConfig(
        name='projectConfigs/{}'.format(int(now.timestamp())),
        create_time=now.isoformat(),
        config=config_dict,
        hash=config_md5_hash)

    self._saved_configs.append((project_config, {}))
    logging.info('Saving a new ProjectConfig to the API: %s',
                 project_config.name)

  def debug_clear(self):
    """Debug method to clear all data saved in the fake API."""
    self._saved_configs = []

  def debug_dump_all(self):
    """Debug method to dump contents of _saved_configs as a string."""
    contents = ['configs:']
    for config, _ in self._saved_configs:
      contents.append('- name: {}'.format(config.name))
      contents.append('  create_time: {}'.format(config.create_time))
      contents.append('  hash: {}'.format(config.hash))

    return '\n'.join(contents)
