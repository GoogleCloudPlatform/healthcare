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
"""Tests for dicom_builder.py."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from typing import Any, Dict, Text
from absl.testing import absltest
import mock
import numpy as np

from hcls_imaging_ml_toolkit import dicom_builder
from hcls_imaging_ml_toolkit import dicom_json
from hcls_imaging_ml_toolkit import tag_values
from hcls_imaging_ml_toolkit import tags
from hcls_imaging_ml_toolkit import test_dwc_util

# Example values of SR Tags used for testing.
_STUDY_UID = '1.2.3'
_UID1 = '4.5.6'
_UID2 = '7.8.9'
_REPORT_CONTENT = 'Test Structured Report.'
_MOCK_STUDY_JSON = test_dwc_util.CreateMockStudyJsonResponse(_STUDY_UID)


class DicomBuilderTest(absltest.TestCase):

  def _AssertJsonTag(self, dicomjson: Dict[Text, Any], tag: tags.DicomTag,
                     expected: Text) -> None:
    """Given a DICOM JSON asserts whether the tag's value is the expected value.

    Args:
      dicomjson: Dict representing the DICOM JSON structure.
      tag: A tuple representing a DICOM Tag and its associated VR.
      expected: String value to compare the value of the tag in the dicomjson
        to.
    """
    self.assertEqual(dicom_json.GetValue(dicomjson, tag), expected)

  @mock.patch.object(dicom_builder, 'GenerateUID', side_effect=[_UID1, _UID2])
  def testBuildJsonSR(self, *_):
    study_json_copy = _MOCK_STUDY_JSON.copy()
    dicom_json_sr = dicom_builder.BuildJsonSR(_REPORT_CONTENT, _MOCK_STUDY_JSON)

    # Ensure the study json is not affected by the builder.
    self.assertEqual(_MOCK_STUDY_JSON, study_json_copy)

    # Test the UIDs, order doesn't matter.
    self.assertCountEqual(
        (dicom_json_sr.series_uid, dicom_json_sr.instance_uid), (_UID1, _UID2))

    dicom_dict = dicom_json_sr.dicom_dict

    # Test Study Level Tags
    for item in _MOCK_STUDY_JSON:
      self.assertIn(item, dicom_dict)
    self._AssertJsonTag(dicom_dict, tags.STUDY_INSTANCE_UID, _STUDY_UID)

    # Test instance level tags.
    self._AssertJsonTag(dicom_dict, tags.SPECIFIC_CHARACTER_SET,
                        dicom_builder._ISO_CHARACTER_SET)
    self._AssertJsonTag(dicom_dict, tags.MEDIA_STORAGE_SOP_INSTANCE_UID,
                        dicom_json_sr.instance_uid)
    self._AssertJsonTag(dicom_dict, tags.MEDIA_STORAGE_SOP_CLASS_UID,
                        tag_values.BASIC_TEXT_SR_CUID)
    self._AssertJsonTag(dicom_dict, tags.SOP_CLASS_UID,
                        tag_values.BASIC_TEXT_SR_CUID)
    self._AssertJsonTag(dicom_dict, tags.MODALITY, tag_values.SR_MODALITY)

    # Test content sequence tags including inference text.
    content_sequence = dicom_json.GetValue(dicom_dict, tags.CONTENT_SEQUENCE)
    self._AssertJsonTag(content_sequence, tags.TEXT_VALUE, _REPORT_CONTENT)
    self._AssertJsonTag(content_sequence, tags.VALUE_TYPE, 'TEXT')
    self._AssertJsonTag(content_sequence, tags.RELATIONSHIP_TYPE, 'CONTAINS')

  @mock.patch.object(dicom_builder, 'GenerateUID', side_effect=[_UID1])
  def testBuildJsonSC(self, *_):
    num_rows = 10
    num_columns = 10
    hufloat = np.ones([num_rows, num_columns, 1])
    image_position_tag_value = '1'
    study_time_tag_value = '2'
    study_instance_uid = '1.2.3.4'
    series_uid = '5.6.7'

    study_dict = {}
    dicom_json.Insert(study_dict, tags.IMAGE_POSITION, image_position_tag_value)
    dicom_json.Insert(study_dict, tags.STUDY_TIME, study_time_tag_value)
    dicom_json.Insert(study_dict, tags.STUDY_INSTANCE_UID, study_instance_uid)
    study_dict_copy = study_dict.copy()
    dicom_json_sc = dicom_builder.BuildJsonSC(hufloat, study_dict, series_uid)

    # Ensure the study_dict is not affected by the builder.
    self.assertEqual(study_dict, study_dict_copy)

    # Test the UIDs.
    self.assertEqual(dicom_json_sc.instance_uid, _UID1)

    dicom_dict = dicom_json_sc.dicom_dict
    # Test study/series level tags.
    self._AssertJsonTag(dicom_dict, tags.STUDY_INSTANCE_UID, study_instance_uid)
    self._AssertJsonTag(dicom_dict, tags.SERIES_INSTANCE_UID, series_uid)
    self._AssertJsonTag(dicom_dict, tags.STUDY_TIME, study_time_tag_value)
    self._AssertJsonTag(dicom_dict, tags.MODALITY, tag_values.OT_MODALITY)

    # Test instance level tags.
    self._AssertJsonTag(dicom_dict, tags.SPECIFIC_CHARACTER_SET,
                        dicom_builder._ISO_CHARACTER_SET)
    self._AssertJsonTag(dicom_dict, tags.MEDIA_STORAGE_SOP_INSTANCE_UID,
                        dicom_json_sc.instance_uid)
    self._AssertJsonTag(dicom_dict, tags.MEDIA_STORAGE_SOP_CLASS_UID,
                        tag_values.SECONDARY_CAPTURE_CUID)
    self._AssertJsonTag(dicom_dict, tags.SOP_CLASS_UID,
                        tag_values.SECONDARY_CAPTURE_CUID)
    self._AssertJsonTag(dicom_dict, tags.TRANSFER_SYNTAX_UID,
                        dicom_builder._IMPLICIT_VR_LITTLE_ENDIAN)

    # Test positional tags.
    self._AssertJsonTag(dicom_dict, tags.IMAGE_POSITION,
                        image_position_tag_value)

    # Test image tags.
    self._AssertJsonTag(dicom_dict, tags.PHOTOMETRIC_INTERPRETATION, 'RGB')
    self._AssertJsonTag(dicom_dict, tags.SAMPLES_PER_PIXEL, 3)
    self._AssertJsonTag(dicom_dict, tags.PLANAR_CONFIGURATION, 0)
    self._AssertJsonTag(dicom_dict, tags.ROWS, num_rows)
    self._AssertJsonTag(dicom_dict, tags.COLUMNS, num_columns)
    self._AssertJsonTag(dicom_dict, tags.BITS_ALLOCATED, 8)
    self._AssertJsonTag(dicom_dict, tags.BITS_STORED, 8)
    self._AssertJsonTag(dicom_dict, tags.HIGH_BIT, 7)
    self._AssertJsonTag(dicom_dict, tags.PIXEL_REPRESENTATION, 0)
    bulkdata_uri = '{}/{}/{}'.format(study_instance_uid, series_uid, _UID1)
    self.assertEqual(dicom_dict[tags.PIXEL_DATA.number]['BulkDataURI'],
                     bulkdata_uri)

    # Test image bulkdata.
    bulkdata = dicom_json_sc.bulkdata_list[0]
    self.assertEqual(bulkdata.uri, bulkdata_uri)
    self.assertEqual(bulkdata.data, hufloat.tobytes())
    self.assertEqual(bulkdata.content_type, 'application/octet-stream')

  @mock.patch.object(
      dicom_builder, 'GenerateUID', side_effect=[_STUDY_UID, _UID1, _UID2])
  def testBuildJsonInstanceFromPng(self, *_):
    sop_class_uid = '1.1.1'
    image = b''
    dicom_json_inst = dicom_builder.BuildJsonInstanceFromPng(
        image, sop_class_uid)
    dicom_dict = dicom_json_inst.dicom_dict

    # Test metadata tags.
    self._AssertJsonTag(dicom_dict, tags.PLANAR_CONFIGURATION, 0)
    self._AssertJsonTag(dicom_dict, tags.PHOTOMETRIC_INTERPRETATION,
                        'MONOCHROME2')
    self._AssertJsonTag(dicom_dict, tags.SOP_CLASS_UID, sop_class_uid)
    self._AssertJsonTag(dicom_dict, tags.STUDY_INSTANCE_UID, _STUDY_UID)
    self._AssertJsonTag(dicom_dict, tags.SERIES_INSTANCE_UID, _UID1)
    self._AssertJsonTag(dicom_dict, tags.SPECIFIC_CHARACTER_SET,
                        dicom_builder._ISO_CHARACTER_SET)
    self._AssertJsonTag(dicom_dict, tags.SOP_INSTANCE_UID, _UID2)
    self._AssertJsonTag(dicom_dict, tags.TRANSFER_SYNTAX_UID,
                        dicom_builder._EXPLICIT_VR_LITTLE_ENDIAN)
    self._AssertJsonTag(dicom_dict, tags.MEDIA_STORAGE_SOP_CLASS_UID,
                        sop_class_uid)
    self._AssertJsonTag(dicom_dict, tags.MEDIA_STORAGE_SOP_INSTANCE_UID, _UID2)
    bulkdata_uri = '{}/{}/{}'.format(_STUDY_UID, _UID1, _UID2)
    self.assertEqual(dicom_dict[tags.PIXEL_DATA.number]['BulkDataURI'],
                     bulkdata_uri)

    # Test image bulkdata.
    bulkdata = dicom_json_inst.bulkdata_list[0]
    self.assertEqual(bulkdata.uri, bulkdata_uri)
    self.assertEqual(bulkdata.data, image)
    self.assertEqual(bulkdata.content_type, 'image/png; transfer-syntax=""')


if __name__ == '__main__':
  absltest.main()
