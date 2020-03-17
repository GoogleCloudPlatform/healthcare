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
"""Tests for dicom_json.py."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest
from hcls_imaging_ml_toolkit import dicom_json
from hcls_imaging_ml_toolkit import dicom_web
from hcls_imaging_ml_toolkit import tags


class DicomJsonTest(absltest.TestCase):

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
    bulkdata = dicom_web.DicomBulkData(
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
