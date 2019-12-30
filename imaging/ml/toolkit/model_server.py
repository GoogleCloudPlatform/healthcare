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
"""Gets predictions from the TensorFlow model server."""

from __future__ import absolute_import
from __future__ import division
from __future__ import google_type_annotations
from __future__ import print_function

from typing import Any, List, Text

import attr
import grpc
from retrying import retry
import tensorflow.google as tf
from google.rpc import code_pb2

from toolkit import exception
from tensorflow_serving.apis import predict_pb2
from tensorflow_serving.apis import prediction_service_pb2_grpc


@attr.s
class Response(object):
  """Response of Tensorflow serving.

  Attributes:
    key: The key that holds the response.
    shape: The expected shape of the response.
  """
  key = attr.ib()  # type: Text
  shape = attr.ib()  # type: List[int]


@attr.s
class ModelConfig(object):
  """Properties of a Tensorflow model.

  Attributes:
    name: Name of the model.
    signature: Signature of the model.
    input_key: The key used to populate the input request.
    response: The expected response structure of the model.
  """
  name = attr.ib()  # type: Text
  signature = attr.ib()  # type: Text
  input_key = attr.ib()  # type: Text
  response = attr.ib()  # type: List[Response]

# Overrwite the max GRPC request and response size to 1GB. The default of 4MB
# is too small for some of the models invoked here.
_MAX_GRPC_REQUEST_AND_RESPONSE_SIZE = 1024 * 1024 * 1024

# gRPC status codes that are retried.
_RETRIABLE_GRPC_STATUS_CODES = (
    grpc.StatusCode.UNAVAILABLE,
    grpc.StatusCode.CANCELLED,
    grpc.StatusCode.RESOURCE_EXHAUSTED,
    grpc.StatusCode.DEADLINE_EXCEEDED,
)


def _IsRetriableGrpcError(e) -> bool:
  """Determines whether the given gRPC exception is retriable.

  Args:
    e: The gRPC exception.

  Returns:
    Whether the exception can be retried.
  """
  return isinstance(e, grpc.Call) and e.code() in _RETRIABLE_GRPC_STATUS_CODES


class ModelServer(object):
  """Gets predictions from the model server."""

  def __init__(self, address: Text, timeout=300.0) -> None:
    """Inits ModelServer with passed args.

    Args:
      address: Address in host:port format.
      timeout: Timeout for prediction calls.
    """
    self._address = address
    self._timeout = timeout

  def Predict(self, input_bytes: bytes,
              model_config: ModelConfig) -> List[List[float]]:
    """Get prediction from the model server.

    Args:
      input_bytes: The input bytes.
      model_config: Configuration for the model to be called.

    Returns:
      The prediction response from TF serving.
    """
    return self._Predict(input_bytes, 1, model_config)

  def PredictExamples(self, examples: List[tf.Example],
                      model_config: ModelConfig) -> List[List[float]]:
    """Get prediction for a list of TF examples from the model server.

    Args:
      examples: The list of TF examples used as input to TF serving.
      model_config: Configuration for the model to be called.

    Returns:
      The prediction response from TF serving.
    """
    serialized_examples = [example.SerializeToString() for example in examples]
    return self._Predict(serialized_examples, len(serialized_examples),
                         model_config)

  def _Predict(self, input_data: Any, num_inputs: int,
               model_config: ModelConfig) -> List[List[float]]:
    """Get prediction from the model server.

    Send the provided input data to the model server over gRPC and returns
    the response.

    Args:
      input_data: Input data fed into the model.
      num_inputs: Number of model inputs (should be set to 1 for non-batch).
      model_config: Configuration for the model to be called.

    Returns:
      The prediction response from TF serving.

    Raises:
      exception.CustomExceptionError: If the shape of response is incorrect or
      if the model server returns an error and it sets the status code to
      INTERNAL.
    """
    try:
      req = predict_pb2.PredictRequest()
      req.model_spec.name = model_config.name
      req.model_spec.signature_name = model_config.signature
      req.inputs[model_config.input_key].CopyFrom(
          tf.make_tensor_proto(input_data, shape=[num_inputs]))
      resp = self._InvokePredictRequest(req)
    except Exception as e:
      raise exception.CustomExceptionError(str(e), code_pb2.Code.INTERNAL)
    floats = []
    for r in model_config.response:
      value = resp.outputs[r.key]
      shape = tf.tensor_util.TensorShapeProtoToList(value.tensor_shape)
      if shape != r.shape:
        raise exception.CustomExceptionError(
            'Model returned invalid shape {}, want {}'.format(shape, r.shape),
            code_pb2.Code.INTERNAL)
      floats.append(value.float_val[:])
    return floats

  @retry(
      retry_on_exception=_IsRetriableGrpcError,
      wait_exponential_multiplier=2000,
      wait_exponential_max=32000,
      stop_max_attempt_number=5)
  def _InvokePredictRequest(
      self, req: predict_pb2.PredictRequest) -> predict_pb2.PredictResponse:
    """Invokes a Predict request to TF serving.

    This function will retry all transient errors.

    Args:
      req: The Predict request.

    Returns:
      The prediction response from TF serving.
    """
    options = [
        ('grpc.max_send_message_length', _MAX_GRPC_REQUEST_AND_RESPONSE_SIZE),
        ('grpc.max_receive_message_length', _MAX_GRPC_REQUEST_AND_RESPONSE_SIZE)
    ]
    channel = grpc.insecure_channel(self._address, options)
    stub = prediction_service_pb2_grpc.PredictionServiceStub(channel)
    return stub.Predict(req, self._timeout)
