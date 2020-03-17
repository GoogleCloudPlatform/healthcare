# Copyright 2019 Google LLC.
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
"""Tests for legacy pubsub_format.py functionality."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import posixpath
from absl.testing import absltest
from absl.testing import parameterized

from google.rpc import code_pb2
from hcls_imaging_ml_toolkit import dicom_web
from hcls_imaging_ml_toolkit import exception
from hcls_imaging_ml_toolkit import pubsub_format
from hcls_imaging_ml_toolkit import test_pubsub_util as tpsu

_DICOM_WEB_PATH = ('projects/project_name/locations/us-central1/datasets/'
                   'dataset_name/dicomStores/dicom_store_name/dicomWeb')
_OUTPUT_DICOM_STORE_PATH = ('projects/project_name/locations/us-central1/'
                            'datasets/dataset_name/dicomStores/'
                            'output_dicom_store_name')
_DICOM_WEB_URL = posixpath.join(dicom_web.CLOUD_HEALTHCARE_API_URL,
                                _DICOM_WEB_PATH)
_ACK_ID = 'ack_id'
_MESSAGE_ID = 'message_id'
_STUDY_ID = '1.2.3'
_SERIES_UID = '4.5.6'
_INSTANCE_UID = '7.9.9'

_STUDY_PATH = posixpath.join(_DICOM_WEB_PATH, 'studies', _STUDY_ID)
_SERIES_PATH = posixpath.join(_STUDY_PATH, 'series', _SERIES_UID)
_INSTANCE_PATH = posixpath.join(_SERIES_PATH, 'instances', _INSTANCE_UID)
_INVALID_PATH = 'invalid_path'


class PubsubFormatLegacyTest(parameterized.TestCase):

  @parameterized.parameters([{}], [{
      'output_dicom_store_path': _OUTPUT_DICOM_STORE_PATH
  }])
  def testExpectedPath(self, attributes):
    """Pub/Sub messages with valid format are parsed."""
    pubsub_message = tpsu.CreatePubsubReceivedMessage(_ACK_ID, _SERIES_PATH,
                                                      _MESSAGE_ID, attributes)
    parsed_message = pubsub_format.ParseMessageLegacy(
        pubsub_message.message,
        pubsub_format.PathTypeLegacy.SERIES)
    self.assertEqual(parsed_message.dicomweb_url, _DICOM_WEB_URL)
    self.assertEqual(parsed_message.study_uid, _STUDY_ID)
    self.assertEqual(parsed_message.series_uid, _SERIES_UID)
    self.assertEqual(parsed_message.output_dicom_store_path,
                     attributes.get('output_dicom_store_path'))

  def testInvalidInputPath(self):
    """Pub/Sub messages with invalid input paths throw exception."""
    pubsub_message = tpsu.CreatePubsubReceivedMessage(_ACK_ID, _INVALID_PATH,
                                                      _MESSAGE_ID)
    with self.assertRaises(exception.CustomExceptionError) as cee:
      pubsub_format.ParseMessageLegacy(pubsub_message.message,
                                       pubsub_format.PathTypeLegacy.SERIES)
    self.assertEqual(cee.exception.status_code, code_pb2.Code.INVALID_ARGUMENT)

  def testInvalidOutputPath(self):
    """Pub/Sub messages with invalid output paths throw exception."""
    pubsub_message = tpsu.CreatePubsubReceivedMessage(
        _ACK_ID, _SERIES_PATH, _MESSAGE_ID,
        {'output_dicom_store_path': _INVALID_PATH})
    with self.assertRaises(exception.CustomExceptionError) as cee:
      pubsub_format.ParseMessageLegacy(pubsub_message.message,
                                       pubsub_format.PathTypeLegacy.SERIES)
    self.assertEqual(cee.exception.status_code, code_pb2.Code.INVALID_ARGUMENT)


if __name__ == '__main__':
  absltest.main()
