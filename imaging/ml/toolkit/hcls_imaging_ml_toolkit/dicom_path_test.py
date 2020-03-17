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
"""Tests for dicom_path.py."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest
from absl.testing import parameterized

from hcls_imaging_ml_toolkit import dicom_path
from hcls_imaging_ml_toolkit import test_dicom_path_util as tdpu


class DicomPathTest(parameterized.TestCase):

  def _AssertStoreAttributes(self, path: dicom_path.Path):
    self.assertEqual(path.project_id, tdpu.PROJECT_NAME)
    self.assertEqual(path.location, tdpu.LOCATION)
    self.assertEqual(path.dataset_id, tdpu.DATASET_ID)
    self.assertEqual(path.store_id, tdpu.STORE_ID)

  def testStorePath(self):
    """Store path is parsed correctly and behaves as expected."""
    store_path = dicom_path.FromString(tdpu.STORE_PATH_STR)
    self._AssertStoreAttributes(store_path)
    self.assertIsNone(store_path.study_uid)
    self.assertIsNone(store_path.series_uid)
    self.assertIsNone(store_path.instance_uid)
    self.assertEqual(store_path.type, dicom_path.Type.STORE)
    self.assertEqual(store_path.dicomweb_path_str, tdpu.DICOMWEB_PATH_STR)
    self.assertEqual(str(store_path), tdpu.STORE_PATH_STR)
    self.assertEqual(str(store_path.GetStorePath()), tdpu.STORE_PATH_STR)

  def testStudyPath(self):
    """Study path is parsed correctly and behaves as expected."""
    study_path = dicom_path.FromString(tdpu.STUDY_PATH_STR)
    self._AssertStoreAttributes(study_path)
    self.assertEqual(study_path.study_uid, tdpu.STUDY_UID)
    self.assertIsNone(study_path.series_uid)
    self.assertIsNone(study_path.instance_uid)
    self.assertEqual(study_path.type, dicom_path.Type.STUDY)
    self.assertEqual(study_path.dicomweb_path_str, tdpu.DICOMWEB_PATH_STR)
    self.assertEqual(str(study_path), tdpu.STUDY_PATH_STR)
    self.assertEqual(str(study_path.GetStorePath()), tdpu.STORE_PATH_STR)
    self.assertEqual(str(study_path.GetStudyPath()), tdpu.STUDY_PATH_STR)

  def testSeriesPath(self):
    """Series path is parsed correctly and behaves as expected."""
    series_path = dicom_path.FromString(tdpu.SERIES_PATH_STR)
    self._AssertStoreAttributes(series_path)
    self.assertEqual(series_path.study_uid, tdpu.STUDY_UID)
    self.assertEqual(series_path.series_uid, tdpu.SERIES_UID)
    self.assertIsNone(series_path.instance_uid)
    self.assertEqual(series_path.type, dicom_path.Type.SERIES)
    self.assertEqual(series_path.dicomweb_path_str, tdpu.DICOMWEB_PATH_STR)
    self.assertEqual(str(series_path), tdpu.SERIES_PATH_STR)
    self.assertEqual(str(series_path.GetStorePath()), tdpu.STORE_PATH_STR)
    self.assertEqual(str(series_path.GetStudyPath()), tdpu.STUDY_PATH_STR)
    self.assertEqual(str(series_path.GetSeriesPath()), tdpu.SERIES_PATH_STR)

  def testInstancePath(self):
    """Instance path is parsed correctly and behaves as expected."""
    instance_path = dicom_path.FromString(tdpu.INSTANCE_PATH_STR)
    self._AssertStoreAttributes(instance_path)
    self.assertEqual(instance_path.study_uid, tdpu.STUDY_UID)
    self.assertEqual(instance_path.series_uid, tdpu.SERIES_UID)
    self.assertEqual(instance_path.instance_uid, tdpu.INSTANCE_UID)
    self.assertEqual(instance_path.type, dicom_path.Type.INSTANCE)
    self.assertEqual(instance_path.dicomweb_path_str, tdpu.DICOMWEB_PATH_STR)
    self.assertEqual(str(instance_path), tdpu.INSTANCE_PATH_STR)
    self.assertEqual(str(instance_path.GetStorePath()), tdpu.STORE_PATH_STR)
    self.assertEqual(str(instance_path.GetStudyPath()), tdpu.STUDY_PATH_STR)
    self.assertEqual(str(instance_path.GetSeriesPath()), tdpu.SERIES_PATH_STR)

  @parameterized.parameters('/', '@')
  def testNoForwardSlashOrAt(self, illegal_char):
    """ValueError is raised when an attribute contains '/' or '@'."""
    self.assertRaises(ValueError, dicom_path.Path, 'project%cid' % illegal_char,
                      'l', 'd', 's')
    self.assertRaises(ValueError, dicom_path.Path, 'p',
                      'locat%cion' % illegal_char, 'd', 's')
    self.assertRaises(ValueError, dicom_path.Path, 'p', 'l',
                      'data%cset' % illegal_char, 's')
    self.assertRaises(ValueError, dicom_path.Path, 'p', 'l', 'd',
                      'st%core' % illegal_char)
    self.assertRaises(ValueError, dicom_path.Path, 'p', 'l', 'd', 's',
                      '1.2%c3' % illegal_char)
    self.assertRaises(ValueError, dicom_path.Path, 'p', 'l', 'd', 's', '1.2.3',
                      '4.5%c6' % illegal_char)
    self.assertRaises(ValueError, dicom_path.Path, 'p', 'l', 'd', 's', '1.2.3',
                      '4.5.6', '7.8%c9' % illegal_char)

  def testUidMissingError(self):
    """ValueError is raised when an expected UID is missing."""
    self.assertRaises(ValueError, dicom_path.Path, 'p', 'l', 'd', 's', None,
                      '4.5.6')
    self.assertRaises(ValueError, dicom_path.Path, 'p', 'l', 'd', 's', 'stuid',
                      None, '7.8.9')

  def testFromStringInvalid(self):
    """ValueError raised when the path string is invalid."""
    self.assertRaises(ValueError, dicom_path.FromString, 'invalid_path')

  def testFromStringTypeError(self):
    """ValueError raised when the expected type doesn't match the actual one."""
    for path_type in dicom_path.Type:
      if path_type != dicom_path.Type.STORE:
        self.assertRaises(ValueError, dicom_path.FromString,
                          tdpu.STORE_PATH_STR, path_type)
      if path_type != dicom_path.Type.STUDY:
        self.assertRaises(ValueError, dicom_path.FromString,
                          tdpu.STUDY_PATH_STR, path_type)
      if path_type != dicom_path.Type.SERIES:
        self.assertRaises(ValueError, dicom_path.FromString,
                          tdpu.SERIES_PATH_STR, path_type)
      if path_type != dicom_path.Type.INSTANCE:
        self.assertRaises(ValueError, dicom_path.FromString,
                          tdpu.INSTANCE_PATH_STR, path_type)


if __name__ == '__main__':
  absltest.main()
