# python3
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
"""Tests for the (place-holder) AuditApiClient."""

import datetime
import textwrap
from absl.testing import absltest

from deploy.audit_backend.processors import audit_api_client
from deploy.audit_backend.processors import audit_api_resources


def get_fake_now():
  """Returns a fake now function that increases by 1 day on each call."""
  next_time = datetime.datetime(2019, 1, 1, 8, 30, 55)

  def now():
    nonlocal next_time
    current_time = next_time
    next_time = current_time + datetime.timedelta(days=1)
    return current_time

  return now


class AuditApiClientTest(absltest.TestCase):

  def setUp(self):
    super(AuditApiClientTest, self).setUp()
    self.api_client = audit_api_client.AuditApiClient(now=get_fake_now())

  def test_get_latest_config_md5_hash(self):
    for i in range(5):
      self.api_client.create_new_project_config({'config_num': i},
                                                'hash_{}'.format(i))
    self.assertEqual('hash_4', self.api_client.get_latest_config_md5_hash())

  def test_get_latest_config_md5_hash_no_configs(self):
    self.assertIsNone(self.api_client.get_latest_config_md5_hash())

  def test_create_new_project_config(self):
    for i in range(5):
      self.api_client.create_new_project_config({'config_num': i},
                                                'hash_{}'.format(i))
    for i in range(5):
      config, scanners = self.api_client._saved_configs[i]
      expected_time = datetime.datetime(2019, 1, 1 + i, 8, 30, 55)
      expected_config = audit_api_resources.ProjectConfig(
          name='projectConfigs/{}'.format(int(expected_time.timestamp())),
          create_time=expected_time.isoformat(),
          config={'config_num': i},
          hash='hash_{}'.format(i))
      self.assertEqual(expected_config, config)
      self.assertEqual({}, scanners)

  def test_debug_clear(self):
    for i in range(5):
      self.api_client.create_new_project_config({'config_num': i},
                                                'hash_{}'.format(i))
    self.api_client.debug_clear()
    self.assertEqual([], self.api_client._saved_configs)

  def test_debug_dump_all(self):
    for i in range(2):
      self.api_client.create_new_project_config({'config_num': i},
                                                'hash_{}'.format(i))
    expected = textwrap.dedent("""\
    configs:
    - name: projectConfigs/1546360255
      create_time: 2019-01-01T08:30:55
      hash: hash_0
    - name: projectConfigs/1546446655
      create_time: 2019-01-02T08:30:55
      hash: hash_1""")
    self.assertEqual(expected, self.api_client.debug_dump_all())


if __name__ == '__main__':
  absltest.main()
