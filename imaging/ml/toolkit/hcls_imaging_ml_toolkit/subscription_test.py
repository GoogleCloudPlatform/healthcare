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
"""Tests for subscription.py."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import logging
import posixpath
from absl.testing import absltest
from absl.testing import parameterized
import mock
import google.auth.credentials
from google.cloud import pubsub_v1

from hcls_imaging_ml_toolkit import subscription
from hcls_imaging_ml_toolkit import test_pubsub_util as tpsu

_ACK_ID = 'ack_id'
_SERIES_PATH = 'series_path'
_MESSAGE_ID = 'message_id'
_PROJECT_ID = 'project_id'
_SUBSCRIPTION_ID = 'subscription_id'
_SUBSCRIPTION_PATH = posixpath.join('projects', _PROJECT_ID, 'subscriptions',
                                    _SUBSCRIPTION_ID)


class SubscriptionTest(parameterized.TestCase):

  def setUp(self):
    super(SubscriptionTest, self).setUp()
    mock.patch.object(pubsub_v1, 'SubscriberClient').start()
    self._sub_client = subscription.SubscriptionClient(
        _PROJECT_ID, _SUBSCRIPTION_ID, credentials=mock.MagicMock())
    self._subscriber = self._sub_client._subscriber
    self._sub_client._subscription_path = _SUBSCRIPTION_PATH

  def tearDown(self):
    super(SubscriptionTest, self).tearDown()
    self._sub_client.Close()

  def testPull(self, *_):
    """Pull returns message when response is received."""
    msg = tpsu.CreatePubsubReceivedMessage(_ACK_ID, _SERIES_PATH, _MESSAGE_ID)
    pull_resp = pubsub_v1.types.PullResponse(received_messages=[msg])
    self._subscriber.pull.return_value = pull_resp

    messages = self._sub_client.Pull()
    self.assertLen(messages, 1)

    self.assertListEqual(self._subscriber.pull.call_args_list, [
        mock.call(
            subscription=_SUBSCRIPTION_PATH,
            max_messages=1,
            return_immediately=False)
    ])
    self.assertEqual(messages[0].ack_id, _ACK_ID)
    self.assertEqual(messages[0].message.message_id, _MESSAGE_ID)
    self.assertEqual(messages[0].message.data.decode('utf-8'), _SERIES_PATH)

  @parameterized.parameters(
      google.api_core.exceptions.RetryError('error_message', Exception),
      google.api_core.exceptions.GoogleAPICallError('error_message', Exception))
  @mock.patch.object(logging, 'exception')
  def testPullLoggedExceptions(self, pull_error, *_):
    """Pull returns empty list and logs exceptions."""
    self._subscriber.pull.side_effect = [pull_error]

    messages = self._sub_client.Pull()
    self.assertEmpty(messages)
    self._subscriber.pull.assert_called_once_with(
        subscription=_SUBSCRIPTION_PATH,
        max_messages=1,
        return_immediately=False)
    self.assertEqual(logging.exception.call_count, 1,
                     'Errors should be logged.')

  @mock.patch.object(logging, 'exception')
  def testPullReturnImmediately(self, *_):
    """Pull passes through |return_immediately|."""
    self._subscriber.pull.side_effect = [
        google.api_core.exceptions.RetryError('error_message', Exception)
    ]

    self._sub_client.Pull(return_immediately=True)
    self.assertEqual(self._subscriber.pull.call_count, 1)
    self.assertListEqual(self._subscriber.pull.call_args_list, [
        mock.call(
            subscription=_SUBSCRIPTION_PATH,
            max_messages=1,
            return_immediately=True)
    ])


if __name__ == '__main__':
  absltest.main()
