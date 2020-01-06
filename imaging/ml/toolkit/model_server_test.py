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
"""Tests for model_server.py."""

from __future__ import absolute_import
from __future__ import division
from __future__ import google_type_annotations
from __future__ import print_function

from typing import List
from absl.testing import absltest
from absl.testing import parameterized

import grpc
import mock
import numpy as np
import tensorflow.google as tf

from toolkit import exception
from toolkit import model_server
from tensorflow_serving.apis import predict_pb2

# Dummy model config used for the test.
_MODEL_CONFIG = model_server.ModelConfig(
    name='model',
    signature='signature',
    input_key='input_key',
    # Shape respresents the scores for top two regions of interest.
    response=[model_server.Response(key='output', shape=[1, 2, 1])])


def _CreatePredictResponse(values: List[float],
                           shape: List[int]) -> predict_pb2.PredictResponse:
  """Returns a PredictResponse used in below unit tests."""
  tensor_proto = tf.make_tensor_proto([], shape=shape, dtype=np.float32)
  tensor_proto.float_val.extend(values)
  resp = predict_pb2.PredictResponse(
      outputs={_MODEL_CONFIG.response[0].key: tensor_proto})
  return resp


class FakeGrpcCall(grpc.RpcError, grpc.Call):
  """Fake implemention of grpc.Call that returns injected error."""

  def __init__(self, return_code):
    """Inits FakeGrpcCall with passed args.

    Args:
      return_code: The code to return for the gRPC call.
    """
    self._return_code = return_code

  def code(self):
    return self._return_code


class ModelServerTest(parameterized.TestCase):

  def setUp(self):
    super(ModelServerTest, self).setUp()
    predictor_stub_mock = mock.patch.object(
        model_server, 'prediction_service_pb2_grpc').start()
    self.addCleanup(predictor_stub_mock.stop)
    self._predict_mock = predictor_stub_mock.PredictionServiceStub().Predict
    self._model_server = model_server.ModelServer('localhost:8500')

  def testPredict(self):
    response = [0, 1]
    self._predict_mock.side_effect = [
        _CreatePredictResponse(response, _MODEL_CONFIG.response[0].shape)
    ]
    resp = self._model_server.Predict(b'', _MODEL_CONFIG)
    self.assertEqual(resp, [response])

  def testPredictExamples(self):
    response = [0, 1]
    self._predict_mock.side_effect = [
        _CreatePredictResponse(response, _MODEL_CONFIG.response[0].shape)
    ]
    example_list = [tf.Example()]
    resp = self._model_server.PredictExamples(example_list, _MODEL_CONFIG)
    self.assertEqual(resp, [response])

  @parameterized.parameters(
      (grpc.StatusCode.UNAVAILABLE), (grpc.StatusCode.CANCELLED),
      (grpc.StatusCode.RESOURCE_EXHAUSTED), (grpc.StatusCode.DEADLINE_EXCEEDED))
  def testPredictWithRetriedTransientFailure(self, code):
    response = [0, 1]
    self._predict_mock.side_effect = [
        FakeGrpcCall(code),
        _CreatePredictResponse(response, _MODEL_CONFIG.response[0].shape)
    ]
    resp = self._model_server.Predict(b'', _MODEL_CONFIG)
    self.assertEqual(resp, [response])

  def testPredictIncorrectShape(self):
    self._predict_mock.side_effect = [_CreatePredictResponse([0, 1], [1])]
    self.assertRaises(exception.CustomExceptionError,
                      self._model_server.Predict, b'', _MODEL_CONFIG)


if __name__ == '__main__':
  absltest.main()
