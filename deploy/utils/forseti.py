# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Forseti provides utilities to manage Forseti instances."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import re

from deploy.utils import runner


_FORSETI_SERVER_SERVICE_ACCOUNT_FILTER = 'email:forseti-server-gcp-*'
_FORSETI_SERVER_BUCKET_RE = re.compile(r'gs://forseti-server-.*')


def get_server_service_account(forseti_project_id):
  """Get the service account for the Forseti server instance.

  Assumes there is only one Forseti instance installed in the project.

  Args:
    forseti_project_id (str): id of the Forseti project.

  Returns:
    str: the forseti server service account.

  Raises:
    ValueError: if gcloud returns an unexpected number of service accounts.
  """
  output = runner.run_gcloud_command([
      'iam', 'service-accounts', 'list',
      '--format', 'value(email)',
      '--filter', _FORSETI_SERVER_SERVICE_ACCOUNT_FILTER,
  ], project_id=forseti_project_id)

  service_accounts = output.strip().split('\n')
  if len(service_accounts) != 1:
    raise ValueError(
        ('Unexpected number of Forseti server service accounts: '
         'got {}, want 1, {}'.format(len(service_accounts), output)))
  return service_accounts[0]


def get_server_bucket(forseti_project_id):
  """Get the bucket holding the Forseti server instance's configuration.

  Args:
    forseti_project_id (str): id of the Forseti project.

  Returns:
    str: the forseti server bucket name.

  Raises:
    ValueError: if failure in finding the bucket in the gsutil command output.
  """
  output = runner.run_command(['gsutil', 'ls', '-p', forseti_project_id],
                              get_output=True)

  match = _FORSETI_SERVER_BUCKET_RE.search(output)
  if not match:
    raise ValueError('Failed to find Forseti server bucket: {}'.format(output))
  return match.group(0)
