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
"""DICOM tags."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from typing import Text
import attr


@attr.s(frozen=True)
class DicomTag(object):
  # Tag number.
  number = attr.ib(type=Text)
  # Tag VR (value representation).
  vr = attr.ib(type=Text)


# List of DICOM tags. Ordered by tag number.
SPECIFIC_CHARACTER_SET = DicomTag(number='00080005', vr='CS')
IMAGE_TYPE = DicomTag(number='00080008', vr='CS')
SOP_CLASS_UID = DicomTag(number='00080016', vr='UI')
SOP_INSTANCE_UID = DicomTag(number='00080018', vr='UI')
STUDY_DATE = DicomTag(number='00080020', vr='DA')
STUDY_TIME = DicomTag(number='00080030', vr='TM')
ACCESSION_NUMBER = DicomTag(number='00080050', vr='SH')
MODALITY = DicomTag(number='00080060', vr='CS')
MANUFACTURER = DicomTag(number='00080070', vr='LO')
REFERRING_PHYSICIAN_NAME = DicomTag(number='00080090', vr='PN')
CODE_VALUE = DicomTag(number='00080100', vr='SH')
CODE_SCHEME_DESIGNATOR = DicomTag(number='00080102', vr='SH')
CODE_SCHEME_VERSION = DicomTag(number='00080103', vr='SH')
CODE_MEANING = DicomTag(number='00080104', vr='LO')
STUDY_DESCRIPTION = DicomTag(number='00081030', vr='LO')
PROCEDURE_CODE_SEQUENCE = DicomTag(number='00081032', vr='SQ')
NAME_OF_PHYSICIAN_READING_STUDY = DicomTag(number='00081060', vr='PN')
ADMITTING_DIAGNOSES_DESCRIPTION = DicomTag(number='00081080', vr='LO')
REFERENCED_STUDY_SEQUENCE = DicomTag(number='00081110', vr='SQ')
REFERENCED_PATIENT_SEQUENCE = DicomTag(number='00081120', vr='SQ')
REFERENCED_SOP_CLASS_UID = DicomTag(number='00081150', vr='UI')
REFERENCED_SOP_INSTANCE_UID = DicomTag(number='00081155', vr='UI')

PATIENT_NAME = DicomTag(number='00100010', vr='PN')
PATIENT_ID = DicomTag(number='00100020', vr='LO')
ISSUER_OF_PATIENT_ID = DicomTag(number='00100021', vr='LO')
PATIENT_BIRTH_DATE = DicomTag(number='00100030', vr='DA')
PATIENT_BIRTH_TIME = DicomTag(number='00100032', vr='TM')
PATIENT_SEX = DicomTag(number='00100040', vr='CS')
OTHER_PATIENT_IDS = DicomTag(number='00101000', vr='LO')
OTHER_PATIENT_NAMES = DicomTag(number='00101001', vr='PN')
PATIENT_AGE = DicomTag(number='00101010', vr='AS')
PATIENT_SIZE = DicomTag(number='00101020', vr='DS')
PATIENT_WEIGHT = DicomTag(number='00101030', vr='DS')
ETHNIC_GROUP = DicomTag(number='00102160', vr='SH')
OCCUPATION = DicomTag(number='00102180', vr='SH')
ADDITIONAL_PATIENT_HISTORY = DicomTag(number='001021B0', vr='LT')
PATIENT_COMMENTS = DicomTag(number='00104000', vr='LT')

SLICE_THICKNESS = DicomTag(number='00180050', vr='DS')
DATE_OF_SECONDARY_CAPTURE = DicomTag(number='00181012', vr='DA')
TIME_OF_SECONDARY_CAPTURE = DicomTag(number='00181014', vr='TM')
SECONDARY_CAPTURE_DEVICE_MANUFACTURER = DicomTag(number='00181016', vr='LO')
SECONDARY_CAPTURE_DEVICE_MANUFACTURERS_MODEL_NAME = DicomTag(
    number='00181018', vr='LO')
SECONDARY_CAPTURE_DEVICE_SOFTWARE_VERSIONS = DicomTag(
    number='00181019', vr='LO')
IMAGER_PIXEL_SPACING = DicomTag(number='00181164', vr='DS')
VIEW_POSITION = DicomTag(number='00185101', vr='CS')

MEDIA_STORAGE_SOP_CLASS_UID = DicomTag(number='00020002', vr='UI')
MEDIA_STORAGE_SOP_INSTANCE_UID = DicomTag(number='00020003', vr='UI')
TRANSFER_SYNTAX_UID = DicomTag(number='00020010', vr='UI')
IMPLEMENTATION_CLASS_UID = DicomTag(number='00020012', vr='UI')
ACQUISITION_NUMBER = DicomTag(number='00200012', vr='IS')
INSTANCE_NUMBER = DicomTag(number='00200013', vr='IS')
POSITION_REFERENCE_INDICATOR = DicomTag(number='00201040', vr='LO')
SLICE_LOCATION = DicomTag(number='00201041', vr='DS')

STUDY_INSTANCE_UID = DicomTag(number='0020000D', vr='UI')
SERIES_INSTANCE_UID = DicomTag(number='0020000E', vr='UI')
STUDY_ID = DicomTag(number='00200010', vr='SH')
PATIENT_ORIENTATION = DicomTag(number='00200020', vr='CS')
IMAGE_POSITION = DicomTag(number='00200032', vr='DS')  # (Patient)
IMAGE_ORIENTATION = DicomTag(number='00200037', vr='DS')
FRAME_OF_REFERENCE_UID = DicomTag(number='00200052', vr='UI')
IMAGE_LATERALITY_ATTRIBUTE = DicomTag(number='00200062', vr='CS')
CONCATENATION_FRAME_OFFSET_NUMBER = DicomTag(number='00209228', vr='UL')

SAMPLES_PER_PIXEL = DicomTag(number='00280002', vr='US')
PHOTOMETRIC_INTERPRETATION = DicomTag(number='00280004', vr='CS')
PLANAR_CONFIGURATION = DicomTag(number='00280006', vr='US')
NUMBER_OF_FRAMES = DicomTag(number='00280008', vr='IS')
FRAME_INCREMENT_POINTER = DicomTag(number='00280009', vr='AT')
ROWS = DicomTag(number='00280010', vr='US')
COLUMNS = DicomTag(number='00280011', vr='US')
PIXEL_SPACING = DicomTag(number='00280030', vr='DS')
BITS_ALLOCATED = DicomTag(number='00280100', vr='US')
BITS_STORED = DicomTag(number='00280101', vr='US')
HIGH_BIT = DicomTag(number='00280102', vr='US')
PIXEL_REPRESENTATION = DicomTag(number='00280103', vr='US')
WINDOW_CENTER = DicomTag(number='00281050', vr='DS')
WINDOW_WIDTH = DicomTag(number='00281051', vr='DS')
RESCALE_INTERCEPT = DicomTag(number='00281052', vr='DS')
RESCALE_SLOPE = DicomTag(number='00281053', vr='DS')

RELATIONSHIP_TYPE = DicomTag(number='0040A010', vr='CS')
VALUE_TYPE = DicomTag(number='0040A040', vr='CS')
TEXT_VALUE = DicomTag(number='0040A160', vr='UT')
CONTENT_SEQUENCE = DicomTag(number='0040A730', vr='SQ')
CONTAINER_IDENTIFIER = DicomTag(number='00400512', vr='LO')
IMAGE_VOLUME_WIDTH = DicomTag(number='00480001', vr='FL')
IMAGE_VOLUME_HEIGHT = DicomTag(number='00480002', vr='FL')
IMAGE_VOLUME_DEPTH = DicomTag(number='00480003', vr='FL')
TOTAL_PIXEL_MATRIX_COLUMNS = DicomTag(number='00480006', vr='UL')
TOTAL_PIXEL_MATRIX_ROWS = DicomTag(number='00480007', vr='UL')
TOTAL_PIXEL_MATRIX_ORIGIN_SEQUENCE = DicomTag(number='00480008', vr='SQ')
SPECIMEN_LABEL_IN_IMAGE = DicomTag(number='00480010', vr='CS')
EXTENDED_DEPTH_OF_FIELD = DicomTag(number='00480012', vr='CS')

REFERENCED_SERIES_SEQUENCE = DicomTag(number='00081115', vr='SQ')

LABEL_TEXT = DicomTag(number='22000002', vr='UT')
BARCODE_VALUE = DicomTag(number='22000005', vr='LT')

PIXEL_DATA = DicomTag(number='7FE00010', vr='OW')

PATIENT_IDENTITY_REMOVED = DicomTag(number='00120062', vr='CS')
DEIDENTIFICATION_METHOD = DicomTag(number='00120063', vr='LO')
