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
"""Main entry for the Audit Dashboard backend App Engine app."""
import logging
import os
import flask

from processors import audit_api_client
from processors import gcs_client
from processors import project_config_processor

logging.getLogger().setLevel(logging.INFO)
app = flask.Flask(__name__)

# These config variables are specified in app.yaml.
_CLOUD_STORAGE_BUCKET = os.environ['CLOUD_STORAGE_BUCKET']
_PROJECT_CONFIG_YAML_PATH = os.environ['PROJECT_CONFIG_YAML_PATH']

# Until the API is launched, keep AuditApiClient instance between requests for
# debugging.
audit_api = audit_api_client.AuditApiClient()


# Handlers for each type of data processor.
@app.route('/processors/update_project_config')
def update_project_config():
  """Handler to update the ProjectConfig from a YAML file on GCS."""
  gcs = gcs_client.GcsClient(_CLOUD_STORAGE_BUCKET)
  yaml_path = _PROJECT_CONFIG_YAML_PATH
  if project_config_processor.update_project_config(yaml_path, gcs, audit_api):
    return 'Updated config from {}'.format(yaml_path)
  else:
    return 'Config not updated.'


# Debug handlers.
# TODO: Remove debug handlers once API is launched.
@app.route('/debug/dump_all_configs')
def dump_all_configs():
  """"Debug path to dump all configs saved in fake Audit API."""
  return audit_api.debug_dump_all()


@app.route('/debug/clear')
def clear_configs():
  """"Debug path to clear all configs saved in fake Audit API."""
  audit_api.debug_clear()
  return 'OK'
