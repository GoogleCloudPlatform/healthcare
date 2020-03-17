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
"""Pub/Sub response messages and handler."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import collections
import enum
import json
from typing import Any, Dict, List, Text, Optional

from absl import logging
import attr

from google.cloud import pubsub_v1
from google.rpc import code_pb2


class InstanceType(enum.Enum):
  """Defines types of DICOM objects created."""

  SR = 'structured_report'
  """Represents a DICOM Structured Report."""

  SC = 'secondary_capture'
  """Represents a DICOM Secondary Capture."""


@attr.s
class CreatedInstance(object):
  """Stores the path and type of the DICOM instance created.

  Attributes:
    path: The instance path.
    type: Type of the created DICOM object.
  """
  path = attr.ib(type=Text)
  type = attr.ib(type=InstanceType)

  def ToDict(self):
    """Returns this object as a json-serializable dictionary."""
    return {'path': self.path, 'type': self.type.value}


@attr.s
class ResponseMessage(object):
  """Stores the response Pub/Sub message.

  Attributes:
    status_code: Status code of the response to the input message.
    input_message_id: Input Pub/Sub message id.
    input_message_data: Input Pub/Sub message data.
    created_instances: List of Dicts representing created instances.
    error_message: Error message if an error occurred.
  """
  status_code = attr.ib(type=code_pb2.Code)
  input_message_id = attr.ib(type=Text)
  input_message_data = attr.ib(type=Text)
  created_instances = attr.ib(
      type=Optional[List[Dict[Text, Any]]], default=None)
  error_message = attr.ib(type=Optional[Text], default=None)

  def ToJson(self) -> bytes:
    """Returns all non-empty fields of object as a json-serialized byte string."""
    return json.dumps(
        attr.asdict(
            self,
            dict_factory=collections.OrderedDict,
            filter=lambda attr, value: value is not None)).encode()


class ResponseHandler(object):
  """Handles response Pub/Sub messages.

     Publishes response Pub/Sub messages to results topic. In cases where the
     inference response is an error it also prints traces and logs to
     stackdriver.
  """

  def __init__(self, publisher_client, monitoring_client):
    """Inits PubSubPublisher with args.

    Args:
      publisher_client: PublisherClient used to publish messages.
      monitoring_client: MonitoringClient used to publish error metrics to
        stackdriver.
    """
    self._publisher_client = publisher_client
    self._monitoring_client = monitoring_client

  def HandleException(self, status_code: code_pb2.Code, message: Text,
                      input_message: pubsub_v1.types.PubsubMessage) -> None:
    """Creates, logs, and publishes a JSON Pub/Sub message.

    Args:
      status_code: Status code of the exception that caused the error.
      message: Error message to be sent.
      input_message: Input Pub/Sub message that caused the error.
    """
    logging.exception(None)
    response_message = ResponseMessage(
        status_code=status_code,
        input_message_id=input_message.message_id,
        input_message_data=input_message.data.decode(),
        error_message=str(message),
    )
    self._PublishMessage(response_message)
    logging.error('Sent error to Pub/Sub: %s', response_message.ToJson())
    self._monitoring_client.WriteErrorMetric(status_code)

  def PublishSuccessResponse(
      self, created_instances: List[CreatedInstance],
      input_message: pubsub_v1.types.PubsubMessage) -> None:
    """Publishes success response Pub/Sub messages to the results topic.

    The success response messages contain a list of the created instances.

    Args:
      created_instances: List of created instances.
      input_message: Input Pub/Sub message which results are for.
    """
    created_instances = [instance.ToDict() for instance in created_instances]
    self._PublishMessage(
        ResponseMessage(
            status_code=code_pb2.Code.OK,
            input_message_id=input_message.message_id,
            input_message_data=input_message.data.decode(),
            created_instances=created_instances,
        ))
    logging.info('Published Success Response Pub/Sub message.')

  def _PublishMessage(self, pubsub_message: ResponseMessage) -> None:
    pubsub_message_json = pubsub_message.ToJson()
    self._publisher_client.Publish(pubsub_message_json)
