# Lint as: python2, python3
# Copyright 2020 Google LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Client library for interacting with Cloud AI Platform(CAIP) Prediction API."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from typing import Any, Dict, Optional, Text

import attr
import google_auth_httplib2
import googleapiclient.discovery
import google.auth


@attr.s
class ModelConfig(object):
  """Properties of a CAIP Prediction model.

  Attributes:
    name: Name of the model.
    output_key: The output key to return.
  """
  name = attr.ib(type=Text)
  output_key = attr.ib(default=None, type=Optional[Text])


class PredictError(Exception):
  """"Exception representing an error from the CAIP Prediction API."""


class Predictor(object):
  """Gets predictions from AI Platform Prediction API."""

  def __init__(self):
    credentials, _ = google.auth.default()
    credentials = google.auth.credentials.with_scopes_if_required(
        credentials, ['https://www.googleapis.com/auth/cloud-platform'])

    # https://developers.google.com/api-client-library/python/guide/thread_safety
    # httplib2.Http (which google_auth_httplib2 wraps) is not threadsafe, so
    # instantiate a new one per request.
    def _BuildRequest(_, *args, **kwargs):
      http = google_auth_httplib2.AuthorizedHttp(credentials)
      return googleapiclient.http.HttpRequest(http, *args, **kwargs)

    self._caip_client = googleapiclient.discovery.build(
        'ml', 'v1', requestBuilder=_BuildRequest)

  def Predict(self, model_input: Dict[Text, Any],
              model_config: ModelConfig) -> Any:
    """Envokes CAIP client predictition and returns the model output.

    Args:
      model_input: The model input json.
      model_config: The model configuration used to invoke CMLE.

    Returns:
      The model output.

    Raises:
      PredictError: If unable to get results from CMLE.
    """
    response = self._caip_client.projects().predict(
        name=model_config.name, body=model_input).execute(num_retries=3)
    if 'error' in response:
      raise PredictError(response['error'])
    predictions = response['predictions']
    if model_config.output_key:
      if model_config.output_key not in predictions:
        raise PredictError('CAIP Prediction output missing %s key' %
                           model_config.output_key)
      return predictions[model_config.output_key]
    return predictions
