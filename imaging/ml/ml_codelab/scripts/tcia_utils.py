# Copyright 2018 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Utility functions for dealing with TCIA data."""

import csv
import io
import httplib2

# Labels files that contain breast density labels and UIDs.
_LABEL_PATHS = [
    "https://wiki.cancerimagingarchive.net/download/attachments/22516629/mass_case_description_train_set.csv",
    "https://wiki.cancerimagingarchive.net/download/attachments/22516629/calc_case_description_train_set.csv",
    "https://wiki.cancerimagingarchive.net/download/attachments/22516629/mass_case_description_test_set.csv",
    "https://wiki.cancerimagingarchive.net/download/attachments/22516629/calc_case_description_test_set.csv",
]
_BREAST_DENSITY_COLUMN = {"breast_density", "breast density"}
_IMAGE_FILE_PATH_COLUMN = {"image file path"}

# Blacklist a set of study UIDs for training. These have duplicate images that
# cause warnings in AutoML.
_BLACKLISTED_STUDY_UIDS = {
    "1.3.6.1.4.1.9590.100.1.2.311909379911326678329538827560440159485",
    "1.3.6.1.4.1.9590.100.1.2.16561647310839362507344536782775923598",
    "1.3.6.1.4.1.9590.100.1.2.166931122911456605424645792320616925399",
    "1.3.6.1.4.1.9590.100.1.2.142065133612748878536601581221344377684",
    "1.3.6.1.4.1.9590.100.1.2.67037509913271775014343029030378766029",
    "1.3.6.1.4.1.9590.100.1.2.70050574512425734439842863973827726605",
    "1.3.6.1.4.1.9590.100.1.2.329070348213711088642769330260826004891",
    "1.3.6.1.4.1.9590.100.1.2.407621803913367629615253812872904106953",
    "1.3.6.1.4.1.9590.100.1.2.170983510513291304519837838253728744250",
    "1.3.6.1.4.1.9590.100.1.2.271834418311297478535408615823692973065",
}


def _GetStudyUIDMaps(has_study_uid=None):
  """Returns a map of Study UID to Series UID and Study UID to label.

  Args:
   has_study_uid: If set, it only returns instances that match this Study UID.

  Returns:
   A Dict of Study UID -> Series UID.
   A Dict of Study UID -> label.
  """

  # Download UIDs for breast density 2 and 3.
  http = httplib2.Http(timeout=60, disable_ssl_certificate_validation=True)
  study_uid_to_series_uid = {}
  study_uid_to_label = {}
  for path in _LABEL_PATHS:
    resp, content = http.request(path, method="GET")
    assert resp.status == 200, "Failed to download label files from: " + path
    r = csv.reader(content.decode("utf-8").splitlines(), delimiter=",")
    header = next(r)
    breast_density_column = -1
    image_file_path_column = -1
    for idx, h in enumerate(header):
      if h in _BREAST_DENSITY_COLUMN:
        breast_density_column = idx
      if h in _IMAGE_FILE_PATH_COLUMN:
        image_file_path_column = idx
    assert breast_density_column != -1, "breast_density column not found"
    assert image_file_path_column != -1, "image file path column not found"
    for row in r:
      density = row[breast_density_column]
      if density != "2" and density != "3":
        continue
      dicom_uids = row[image_file_path_column].split("/")
      study_instance_uid, series_instance_uid = dicom_uids[1], dicom_uids[2]
      if study_instance_uid in _BLACKLISTED_STUDY_UIDS:
        continue
      if has_study_uid and has_study_uid != study_instance_uid:
        continue
      study_uid_to_series_uid[study_instance_uid] = series_instance_uid
      study_uid_to_label[study_instance_uid] = density
  return study_uid_to_series_uid, study_uid_to_label


def GetStudyUIDToSeriesUIDMap(has_study_uid=None):
  """Returns a map of Study UID to Series UID.

  Args:
   has_study_uid: If set, it only returns instances that match this Study UID.

  Returns:
   A Dict of Study UID -> Series UID.
  """
  return _GetStudyUIDMaps(has_study_uid)[0]


def GetStudyUIDToLabelMap(has_study_uid=None):
  """Returns a map of Study UID to label.

  Args:
   has_study_uid: If set, it only returns instances that match this Study UID.

  Returns:
   A Dict of Study UID -> label.
  """
  return _GetStudyUIDMaps(has_study_uid)[1]
