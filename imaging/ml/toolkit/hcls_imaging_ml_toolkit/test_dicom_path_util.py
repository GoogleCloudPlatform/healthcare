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
"""Utility class tests using functionality related to DICOMweb paths."""
# pylint: mode=test

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import posixpath

PROJECT_NAME = 'project_name'
LOCATION = 'us-central1'
DATASET_ID = 'dataset-id'
STORE_ID = 'dicom_store-id'
# Support alphanumeric characters in DICOM UIDs
STUDY_UID = '1.2.3a'
SERIES_UID = '4.5.6b'
INSTANCE_UID = '7.8.9c'
DATASET_PATH_STR = posixpath.join('projects', PROJECT_NAME, 'locations',
                                  LOCATION, 'datasets', DATASET_ID)
STORE_PATH_STR = posixpath.join(DATASET_PATH_STR, 'dicomStores', STORE_ID)
DICOMWEB_PATH_STR = posixpath.join(STORE_PATH_STR, 'dicomWeb')
STUDY_PATH_STR = posixpath.join(DICOMWEB_PATH_STR, 'studies', STUDY_UID)
SERIES_PATH_STR = posixpath.join(STUDY_PATH_STR, 'series', SERIES_UID)
INSTANCE_PATH_STR = posixpath.join(SERIES_PATH_STR, 'instances', INSTANCE_UID)
