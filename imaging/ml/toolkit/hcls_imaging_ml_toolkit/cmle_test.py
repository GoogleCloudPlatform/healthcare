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
"""Tests for cmle.py."""
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest
from absl.testing import parameterized
import googleapiclient
import mock
import google.auth

from hcls_imaging_ml_toolkit import cmle

_SAMPLE_MODEL_INPUT_JSON = {
    'instances': [{
        'b64': 'input_data'
    }],
}


class CmleTest(parameterized.TestCase):

  def setUp(self):
    super(CmleTest, self).setUp()
    self.addCleanup(mock.patch.stopall)
    mock.patch.object(
        google.auth.credentials, 'with_scopes_if_required',
        autospec=True).start()
    mock.patch.object(
        google.auth,
        'default',
        return_value=(mock.MagicMock(), mock.MagicMock()),
        autospec=True).start()
    self._discovery_client = mock.patch.object(
        googleapiclient.discovery, 'build', autospec=True).start()
    self._predictor = cmle.Predictor()

  def _SetPredictResponse(self, return_value):
    (self._discovery_client.return_value.projects.return_value.predict
     .return_value.execute.return_value) = return_value

  @parameterized.parameters(
      ({
          'predictions': {
              'key': 'value'
          }
      }, 'key'),
      ({
          'predictions': 'value'
      }, None),
  )
  def testPredictSuccess(self, response, key):
    self._SetPredictResponse(response)

    output = self._predictor.Predict(
        _SAMPLE_MODEL_INPUT_JSON,
        cmle.ModelConfig(name='projects/p1/models/m1', output_key=key))
    self.assertEqual(output, 'value')

  @parameterized.parameters(
      ({
          'error': 'text'
      },),
      ({
          'predictions': {
              'missing_key': 'value'
          }
      },),
  )
  def testPredictError(self, mock_response):
    self._SetPredictResponse(mock_response)
    with self.assertRaises(cmle.PredictError):
      self._predictor.Predict(
          _SAMPLE_MODEL_INPUT_JSON,
          cmle.ModelConfig(name='projects/p1/models/m1', output_key='key'))


if __name__ == '__main__':
  absltest.main()
