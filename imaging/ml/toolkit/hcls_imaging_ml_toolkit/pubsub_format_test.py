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
"""Tests for pubsub_format.py."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import posixpath
from absl.testing import absltest
from absl.testing import parameterized

from google.rpc import code_pb2
from hcls_imaging_ml_toolkit import dicom_path
from hcls_imaging_ml_toolkit import exception
from hcls_imaging_ml_toolkit import pubsub_format
from hcls_imaging_ml_toolkit import test_dicom_path_util as tdpu
from hcls_imaging_ml_toolkit import test_pubsub_util as tpsu

_OUTPUT_STORE_ID = 'output_dicom_store_id'
_OUTPUT_DICOM_STORE_PATH = posixpath.join(tdpu.DATASET_PATH_STR, 'dicomStores',
                                          _OUTPUT_STORE_ID)
_ACK_ID = 'ack_id'
_MESSAGE_ID = 'message_id'
# dicomstores instead of dicomStores.
_INVALID_STORE_PATH = ('projects/project_name/locations/us-central1/datasets/'
                       'dataset_name/dicomstores/store_id')
# sries instead of series.
_INVALID_SERIES_PATH = ('projects/project_name/locations/us-central1/datasets/'
                        'dataset_name/dicomStores/store_id/dicomWeb/studies/'
                        '1.2.3/sries/4.5.6')


class PubsubFormatTest(parameterized.TestCase):

  @parameterized.parameters([{}], [{
      'output_dicom_store_path': _OUTPUT_DICOM_STORE_PATH
  }])
  def testExpectedPath(self, attributes):
    """Pub/Sub messages with valid format are parsed."""
    pubsub_message = tpsu.CreatePubsubReceivedMessage(_ACK_ID,
                                                      tdpu.SERIES_PATH_STR,
                                                      _MESSAGE_ID, attributes)
    parsed_message = pubsub_format.ParseMessage(pubsub_message.message,
                                                dicom_path.Type.SERIES)
    self.assertEqual(str(parsed_message.input_path), tdpu.SERIES_PATH_STR)
    if attributes.get('output_dicom_store_path'):
      self.assertEqual(
          str(parsed_message.output_dicom_store_path),
          attributes.get('output_dicom_store_path'))
    else:
      self.assertIsNone(parsed_message.output_dicom_store_path)

  def testInvalidInputPath(self):
    """Pub/Sub messages with invalid input paths throw exception."""
    pubsub_message = tpsu.CreatePubsubReceivedMessage(_ACK_ID,
                                                      _INVALID_SERIES_PATH,
                                                      _MESSAGE_ID)
    with self.assertRaises(exception.CustomExceptionError) as cee:
      pubsub_format.ParseMessage(pubsub_message.message, dicom_path.Type.SERIES)
    self.assertEqual(cee.exception.status_code, code_pb2.Code.INVALID_ARGUMENT)

  def testInvalidOutputPath(self):
    """Pub/Sub messages with invalid output paths throw exception."""
    pubsub_message = tpsu.CreatePubsubReceivedMessage(
        _ACK_ID, tdpu.SERIES_PATH_STR, _MESSAGE_ID,
        {'output_dicom_store_path': _INVALID_STORE_PATH})
    with self.assertRaises(exception.CustomExceptionError) as cee:
      pubsub_format.ParseMessage(pubsub_message.message, dicom_path.Type.SERIES)
    self.assertEqual(cee.exception.status_code, code_pb2.Code.INVALID_ARGUMENT)


if __name__ == '__main__':
  absltest.main()
