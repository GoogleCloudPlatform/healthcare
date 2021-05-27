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
"""Utility class for tests using Pub/Sub-related functionality."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from typing import Dict, Optional, Text
from google.cloud import pubsub_v1


class PubsubRunLoopExitError(BaseException):
  """Forces exit from the infinite PubsubListener run loop.

  PubsubListener catches all exceptions inheriting from Exception within its
  Run loop. Use this exceptions within tests to force exit.
  """
  pass


def CreatePubsubReceivedMessage(
    ack_id: Text,
    data: Text,
    message_id: Text,
    attributes: Optional[Dict[Text, Text]] = None
) -> pubsub_v1.types.ReceivedMessage:
  """Creates a ReceivedMessage instance for testing.

  Args:
    ack_id: Pubsub ACK ID.
    data: The payload of the Pubsub message.
    message_id: Pubsub Message ID
    attributes: Pubsub attributes.

  Returns:
    Instance of ReceivedMessage.
  """
  return pubsub_v1.types.ReceivedMessage(
      ack_id=ack_id,
      message=pubsub_v1.types.PubsubMessage(
          data=data.encode('utf8'),
          message_id=message_id,
          attributes=attributes))
