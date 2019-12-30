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
"""Utility class tests using DicomWebClient-related functionality."""
# pylint: mode=test

from __future__ import absolute_import
from __future__ import division
from __future__ import google_type_annotations
from __future__ import print_function

from typing import Any, Dict, Text

from toolkit import dicom_builder
from toolkit import dicom_json
from toolkit import tag_values
from toolkit import tags

# Example values of Study Level Tags in the SR used for testing callback.
_ACCESSION_NUMBER = 'Accession Number'
_STUDY_ID = 'Study ID'
_STUDY_DATE = '20000101'
_STUDY_TIME = '235959'
_PATIENT_BIRTH_DATE = '19990101'
_PATIENT_BIRTH_TIME = '235959'
_STUDY_DESCRIPTION = 'Description'
_REFERRING_PHYSICIAN_NAME = 'Physician Name'
_PATIENT_NAME = 'Patient Name'
_PATIENT_ID = 'Patient ID'
_PATIENT_SEX = 'M'
_PATIENT_AGE = '018M'
_PATIENT_SIZE = '99'
_PATIENT_WEIGHT = '99'
_NAME_OF_PHYSICIAN_READING = 'John Doe'
_ADMITTING_DIAGNOSES_DESCRIPTION = 'Description'
_ISSUER_OF_PATIENT_ID = 'Issuer'
_OTHER_PATIENT_IDS = 'Addition Identifier'
_OTHER_PATIENT_NAMES = 'Second Name'
_ETHNIC_GROUP = 'Ethnicity'
_OCCUPATION = 'Job Title'
_ADDITIONAL_PATIENT_HISTORY = 'Additional History'
_PATIENT_COMMENTS = 'Patient Comments'
_CODE_VALUE = 'Code Value'
_CODE_SCHEME_DESIGNATOR = 'Designator'
_CODE_SCHEME_VERSION = 'Version'
_CODE_MEANING = 'Code Meaning'
_REFERENCED_SOP_CLASS_UID = 'Class UID'
_REFERENCED_SOP_INSTANCE_UID = 'Instance UID'


def CreateMockStudyJsonResponse(study_uid: Text) -> Dict[Text, Any]:
  """Creates a Dict representing a DICOM JSON response to a Study Level QIDO."""
  study_json = {}
  dicom_json.Insert(study_json, tags.ACCESSION_NUMBER, _ACCESSION_NUMBER)
  dicom_json.Insert(study_json, tags.STUDY_INSTANCE_UID, study_uid)
  dicom_json.Insert(study_json, tags.STUDY_ID, _STUDY_ID)
  dicom_json.Insert(study_json, tags.STUDY_DATE, _STUDY_DATE)
  dicom_json.Insert(study_json, tags.STUDY_DESCRIPTION, _STUDY_DESCRIPTION)
  dicom_json.Insert(study_json, tags.REFERRING_PHYSICIAN_NAME,
                    _REFERRING_PHYSICIAN_NAME)
  dicom_json.Insert(study_json, tags.STUDY_TIME, _STUDY_TIME)
  dicom_json.Insert(study_json, tags.PATIENT_BIRTH_DATE, _PATIENT_BIRTH_DATE)
  dicom_json.Insert(study_json, tags.PATIENT_BIRTH_TIME, _PATIENT_BIRTH_TIME)
  dicom_json.Insert(study_json, tags.PATIENT_NAME, _PATIENT_NAME)
  dicom_json.Insert(study_json, tags.PATIENT_ID, _PATIENT_ID)
  dicom_json.Insert(study_json, tags.PATIENT_SEX, _PATIENT_SEX)
  dicom_json.Insert(study_json, tags.PATIENT_AGE, _PATIENT_AGE)
  dicom_json.Insert(study_json, tags.PATIENT_SIZE, _PATIENT_SIZE)
  dicom_json.Insert(study_json, tags.PATIENT_WEIGHT, _PATIENT_WEIGHT)
  dicom_json.Insert(study_json, tags.NAME_OF_PHYSICIAN_READING_STUDY,
                    _NAME_OF_PHYSICIAN_READING)
  dicom_json.Insert(study_json, tags.ADMITTING_DIAGNOSES_DESCRIPTION,
                    _ADMITTING_DIAGNOSES_DESCRIPTION)
  dicom_json.Insert(study_json, tags.ISSUER_OF_PATIENT_ID,
                    _ISSUER_OF_PATIENT_ID)
  dicom_json.Insert(study_json, tags.OTHER_PATIENT_IDS, _OTHER_PATIENT_IDS)
  dicom_json.Insert(study_json, tags.OTHER_PATIENT_NAMES, _OTHER_PATIENT_NAMES)
  dicom_json.Insert(study_json, tags.ETHNIC_GROUP, _ETHNIC_GROUP)
  dicom_json.Insert(study_json, tags.OCCUPATION, _OCCUPATION)
  dicom_json.Insert(study_json, tags.ADDITIONAL_PATIENT_HISTORY,
                    _ADDITIONAL_PATIENT_HISTORY)
  dicom_json.Insert(study_json, tags.PATIENT_COMMENTS, _PATIENT_COMMENTS)

  procedure_sequence = {}
  dicom_json.Insert(procedure_sequence, tags.CODE_VALUE, _CODE_VALUE)
  dicom_json.Insert(procedure_sequence, tags.CODE_SCHEME_DESIGNATOR,
                    _CODE_SCHEME_DESIGNATOR)
  dicom_json.Insert(procedure_sequence, tags.CODE_SCHEME_VERSION,
                    _CODE_SCHEME_VERSION)
  dicom_json.Insert(procedure_sequence, tags.CODE_MEANING, _CODE_MEANING)
  dicom_json.Insert(study_json, tags.PROCEDURE_CODE_SEQUENCE,
                    procedure_sequence)

  referenced_study_sequence = {}
  dicom_json.Insert(referenced_study_sequence, tags.REFERENCED_SOP_CLASS_UID,
                    _REFERENCED_SOP_CLASS_UID)
  dicom_json.Insert(referenced_study_sequence, tags.REFERENCED_SOP_INSTANCE_UID,
                    _REFERENCED_SOP_INSTANCE_UID)
  dicom_json.Insert(study_json, tags.REFERENCED_STUDY_SEQUENCE,
                    referenced_study_sequence)

  referenced_patient_sequence = {}
  dicom_json.Insert(referenced_patient_sequence, tags.REFERENCED_SOP_CLASS_UID,
                    _REFERENCED_SOP_CLASS_UID)
  dicom_json.Insert(referenced_patient_sequence,
                    tags.REFERENCED_SOP_INSTANCE_UID,
                    _REFERENCED_SOP_INSTANCE_UID)
  dicom_json.Insert(study_json, tags.REFERENCED_PATIENT_SEQUENCE,
                    referenced_patient_sequence)
  return study_json


# Values for Structured Report.
_STUDY_UID = 'study_uid'
_SR_INSTANCE_UID = 'sr_instance_uid'
_SR_SERIES_UID = 'sr_series_uid'


def CreateMockSRDict(sr_text: Text) -> Dict[Text, Any]:
  """Creates a DICOM JSON dictionary representing a structured report."""
  sr_dict = CreateMockStudyJsonResponse(_STUDY_UID)
  dicom_json.Insert(sr_dict, tags.SOP_CLASS_UID, tag_values.BASIC_TEXT_SR_CUID)
  dicom_json.Insert(sr_dict, tags.MODALITY, tag_values.SR_MODALITY)
  dicom_json.Insert(sr_dict, tags.SERIES_INSTANCE_UID, _SR_SERIES_UID)
  dicom_json.Insert(sr_dict, tags.SPECIFIC_CHARACTER_SET,
                    dicom_builder._ISO_CHARACTER_SET)
  dicom_json.Insert(sr_dict, tags.SOP_INSTANCE_UID, _SR_INSTANCE_UID)

  content_json = {}
  dicom_json.Insert(content_json, tags.RELATIONSHIP_TYPE, 'CONTAINS')
  dicom_json.Insert(content_json, tags.VALUE_TYPE, 'TEXT')
  dicom_json.Insert(content_json, tags.TEXT_VALUE, sr_text)
  dicom_json.Insert(sr_dict, tags.CONTENT_SEQUENCE, content_json)

  dicom_json.Insert(sr_dict, tags.TRANSFER_SYNTAX_UID,
                    dicom_builder._IMPLICIT_VR_LITTLE_ENDIAN)
  dicom_json.Insert(sr_dict, tags.MEDIA_STORAGE_SOP_CLASS_UID,
                    tag_values.BASIC_TEXT_SR_CUID)
  dicom_json.Insert(sr_dict, tags.MEDIA_STORAGE_SOP_INSTANCE_UID,
                    _SR_INSTANCE_UID)
  return sr_dict
