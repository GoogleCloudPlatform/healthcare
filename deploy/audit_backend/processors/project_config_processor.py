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
"""Job for uploading new project configurations to the audit API."""
import logging
import yaml
from google.cloud import exceptions


def update_project_config(config_path, gcs_client, api_client):
  """Updates the project config at the specified path.

  Reads the latest config version from the API and specified GCS path. If the
  md5_hash has changed, then create a new project config version in the audit
  API.

  Args:
    config_path (string): The GCS object containing the project config YAML.
    gcs_client (GcsClient): Client used to read YAML config files from GCS.
    api_client (AuditApiClient): Client used to read and write Project Configs
      to the HCLS Audit API.

  Returns:
    Boolean: True if a new project config version was saved, False otherwise.

  """
  try:
    api_config_version = api_client.get_latest_config_md5_hash()
  except exceptions.GoogleCloudError as e:
    logging.error('Error accessing Audit API: %s', e)
    return False

  gcs_config_blob = gcs_client.get_blob(config_path)
  if not gcs_config_blob:
    logging.error('Error reading config file from GCS: %s', config_path)
    return False

  project_config_md5_hash = gcs_config_blob.md5_hash
  if api_config_version == project_config_md5_hash:
    logging.info('Project Config already up to date.')
    return False

  # Project config file changed, so download and update in the API.
  try:
    project_config_dict = yaml.safe_load(gcs_config_blob.download_as_string())
  except yaml.YAMLError as e:
    logging.error('Error parsing config file: %s', e)
    return False
  except exceptions.GoogleCloudError as e:
    logging.error('Error downloading config file: %s', e)
    return False

  if not project_config_dict:
    # Config file is empty, or only contains comments.
    # TODO: Add schema check.
    logging.error('Bad project config file.')
    return False

  try:
    api_client.create_new_project_config(project_config_dict,
                                         project_config_md5_hash)
  except exceptions.GoogleCloudError as e:
    logging.error('Error saving new project config: %s', e)
    return False
  return True
