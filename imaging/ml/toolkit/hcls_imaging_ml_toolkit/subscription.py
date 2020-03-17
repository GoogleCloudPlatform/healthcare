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
"""Cloud Pub/Sub Subscription client."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import logging
from typing import List, Optional, Text

import google.api_core.exceptions
import google.auth.credentials
from google.cloud import pubsub_v1


class SubscriptionClient(object):
  """Pub/Sub subscription client using google.cloud.pubsub_v1."""

  def __init__(
      self,
      project_id: Text,
      subscription_id: Text,
      credentials: Optional[google.auth.credentials.Credentials] = None
  ) -> None:
    """Inits SubscriptionClient with passed args.

    Args:
      project_id: Project ID containing the Pub/Sub subscription.
      subscription_id: Subscription ID of the Pub/Sub subscription.
      credentials: Google Auth Credentials, if not set it will use the default
        credentials
    """
    self._subscriber = pubsub_v1.SubscriberClient(credentials=credentials)
    self._subscription_path = self._subscriber.subscription_path(
        project_id, subscription_id)

  def Pull(
      self,
      return_immediately: bool = False
  ) -> List[pubsub_v1.types.ReceivedMessage]:
    """Pulls subscription for received Pub/Sub messages.

    Polls until a message is received. If an underlying library raises an
    exception, this will log or ignore the exception and contine to poll. To
    disable polling set |return_immediately| to True.

    Args:
      return_immediately: Flag to return after a single pull response.

    Returns:
      List of Pub/Sub messages received from subscription.
    """
    try:
      return self._subscriber.pull(
          self._subscription_path,
          max_messages=1,
          return_immediately=return_immediately).received_messages
    except (google.api_core.exceptions.GoogleAPICallError,
            google.api_core.exceptions.RetryError):
      logging.exception('Pub/Sub pull error.')
    return []

  def Acknowledge(self, ack_id: Text) -> None:
    self._subscriber.acknowledge(self._subscription_path, [ack_id])

  def ModifyAckDeadline(self, ack_id: Text, ack_deadline_seconds: int) -> None:
    self._subscriber.modify_ack_deadline(self._subscription_path, [ack_id],
                                         ack_deadline_seconds)

  def Close(self) -> None:
    self._subscriber.api.transport.channel.close()
