# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the 'License');
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an 'AS IS' BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Tests for dicom_json.py."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest
from absl.testing import parameterized
import frozendict

from hcls_imaging_ml_toolkit import dicom_json
from hcls_imaging_ml_toolkit import tags


_DICOM_JSON = frozendict.frozendict({
    tags.PATIENT_AGE.number: {
        'vr': tags.PATIENT_AGE.vr,
        'Value': ['049Y']
    },
    tags.SPECIFIC_CHARACTER_SET.number: {
        'vr': tags.SPECIFIC_CHARACTER_SET.vr,  # Missing 'Value' key.
    },
    tags.WINDOW_WIDTH.number: {
        'vr': tags.WINDOW_WIDTH.vr,
        'Value': []  # 'Value' key maps to empty list.
    },
    tags.PATIENT_COMMENTS.number: {
        'vr': tags.PATIENT_COMMENTS.vr,
        'Value': ['abc', 'def', 'ghi']
    },
})


class DicomJsonTest(parameterized.TestCase):

  @parameterized.named_parameters(
      ('SingleValue', tags.PATIENT_AGE, ['049Y']),
      ('MissingValue', tags.SPECIFIC_CHARACTER_SET, None),
      ('EmptyList', tags.WINDOW_WIDTH, []),
      ('MultipleValues', tags.PATIENT_COMMENTS, ['abc', 'def', 'ghi']),
  )
  def testGetList(self, tag, expected_value):
    """Tests expected return values for `GetList()`."""
    actual_value = dicom_json.GetList(_DICOM_JSON, tag)
    self.assertEqual(actual_value, expected_value)

  @parameterized.named_parameters(
      ('SingleValue', tags.PATIENT_AGE, '049Y'),
      ('MissingValue', tags.SPECIFIC_CHARACTER_SET, None),
      ('EmptyList', tags.WINDOW_WIDTH, None),
      ('MultipleValues', tags.PATIENT_COMMENTS, 'abc'),
  )
  def testGetValue(self, tag, expected_value):
    """Tests expected return values for `GetValue()`."""
    actual_value = dicom_json.GetValue(_DICOM_JSON, tag)
    self.assertEqual(actual_value, expected_value)

  def testObjectWithBulkData(self):
    """Tests methods of ObjectWithBulkData class."""
    # Construct dicom_json.ObjectWithBulkData object
    dicom_dict = dict()
    instance_uid = 'instance_uid'
    study_uid = 'study_uid'
    series_uid = 'series_uid'
    dicom_json.Insert(dicom_dict, tags.SOP_INSTANCE_UID, instance_uid)
    dicom_json.Insert(dicom_dict, tags.STUDY_INSTANCE_UID, study_uid)
    dicom_json.Insert(dicom_dict, tags.SERIES_INSTANCE_UID, series_uid)
    bulkdata = dicom_json.DicomBulkData(
        uri='uri',
        data=bytearray('image_array', encoding='utf8'),
        content_type='type')
    bulkdata_list = [bulkdata]
    dicom_object = dicom_json.ObjectWithBulkData(dicom_dict, bulkdata_list)

    # Test properties
    self.assertEqual(dicom_object.instance_uid, instance_uid)
    self.assertEqual(dicom_object.series_uid, series_uid)
    self.assertEqual(dicom_object.study_uid, study_uid)


if __name__ == '__main__':
  absltest.main()
