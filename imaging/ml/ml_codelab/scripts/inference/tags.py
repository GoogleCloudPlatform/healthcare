"""DICOM tags."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import attr


@attr.s
class DicomTag(object):
  # Tag number.
  number = attr.ib()  # type: int
  # Tag VR (value representation).
  vr = attr.ib()  # type: str


# List of DICOM tags.
SOP_CLASS_UID = DicomTag(number='00080016', vr='UI')
SOP_INSTANCE_UID = DicomTag(number='00080018', vr='UI')
STUDY_INSTANCE_UID = DicomTag(number='0020000D', vr='UI')
SERIES_INSTANCE_UID = DicomTag(number='0020000E', vr='UI')
MEDIA_STORAGE_SOP_CLASS_UID = DicomTag(number='00020002', vr='UI')
MEDIA_STORAGE_SOP_INSTANCE_UID = DicomTag(number='00020003', vr='UI')
TRANSFER_SYNTAX_UID = DicomTag(number='00020010', vr='UI')
IMPLEMENTATION_CLASS_UID = DicomTag(number='00020012', vr='UI')
RELATIONSHIP_TYPE = DicomTag(number='0040A010', vr='CS')
VALUE_TYPE = DicomTag(number='0040A040', vr='CS')
TEXT_VALUE = DicomTag(number='0040A160', vr='UT')
CONTENT_SEQUENCE = DicomTag(number='0040A730', vr='SQ')
SPECIFIC_CHARACTER_SET = DicomTag(number='00080005', vr='CS')
MODALITY_TAG = DicomTag(number='00080060', vr='CS')
