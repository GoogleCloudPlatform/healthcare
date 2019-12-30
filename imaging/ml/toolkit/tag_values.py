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
"""Common values of DICOM tags."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

# SOP Class UID for Basic Text Structured Reports.
BASIC_TEXT_SR_CUID = '1.2.840.10008.5.1.4.1.1.88.11'

# SOP Class UID for Secondary Capture.
SECONDARY_CAPTURE_CUID = '1.2.840.10008.5.1.4.1.1.7'

# Modality values.
CT_MODALITY = 'CT'
SR_MODALITY = 'SR'
OT_MODALITY = 'OT'  # Other, used for Secondary Capture objects.
MG_MODALITY = 'MG'
