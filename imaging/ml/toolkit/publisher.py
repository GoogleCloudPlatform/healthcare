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
"""Cloud Pub/Sub Publisher client."""

from __future__ import absolute_import
from __future__ import division
from __future__ import google_type_annotations
from __future__ import print_function

from typing import Dict, Optional, Text


import google.api_core.future
import google.auth.credentials
from google.cloud import pubsub_v1


class PublisherClient(object):
  """Pub/Sub publisher client using google.cloud.pubsub_v1."""

  def __init__(
      self,
      project_id: Text,
      topic_name: Text,
      credentials: Optional[google.auth.credentials.Credentials] = None
  ) -> None:
    """Inits PublisherClient with passed args.

    Args:
      project_id: Project ID containing the Pub/Sub subscription.
      topic_name: Pub/Sub topic name being published to.
      credentials: Google Auth Credentials, if not set it will use the default
        credentials
    """
    self._publisher = pubsub_v1.PublisherClient(credentials=credentials)
    self._topic = self._publisher.topic_path(project_id, topic_name)

  def Publish(
      self,
      data: Text,
      attributes: Optional[Dict[Text, Text]] = None
  ) -> google.api_core.future.Future:
    """Publishes Pub/Sub message to topic set in PublisherClient.

    Args:
      data: Text to be used as body of Pub/Sub message.
      attributes: Optional Dict representing Pub/Sub message attributes and
        their respective values.

    Returns:
      A google.api_core.future.Future running the publishing of the Pub/Sub
      message.
    """
    attributes = attributes if attributes else {}
    return self._publisher.publish(self._topic, data=data, **attributes)
