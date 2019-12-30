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
"""Utility class for building Basic Text DICOM Structured Reports."""

from __future__ import absolute_import
from __future__ import division
from __future__ import google_type_annotations
from __future__ import print_function

import datetime
import os
from typing import Any, Dict, Text
import uuid

from absl import logging
import numpy as np

from toolkit import dicom_json
from toolkit import dicom_web
from toolkit import tag_values
from toolkit import tags

# Google UUID prefix for DICOM objects containing inference results.
_UUID_INFERENCE_PREFIX = '1.3.6.1.4.1.11129.5.3'

# DICOM header preamble is 128-byte long.
_PREAMBLE_LENGTH = 128
# Little Endian Transfer Syntax.
_IMPLICIT_VR_LITTLE_ENDIAN = '1.2.840.10008.1.2'
_EXPLICIT_VR_LITTLE_ENDIAN = '1.2.840.10008.1.2.1'
# Accepted character set.
_ISO_CHARACTER_SET = 'ISO_IR 192'


def BuildJsonSR(
    report_text: Text,
    metadata_json: Dict[Text, Any]) -> dicom_json.ObjectWithBulkData:
  """Builds and returns a Basic Text DICOM JSON Structured Report instance.

  This function will create a new DICOM series.

  Args:
    report_text: Text string to use for the Basic Text DICOM SR.
    metadata_json: Dict of tags (including study-level information) to add.

  Returns:
    DICOM JSON Object containing the Structured Report.
  """
  # Dicom StowJsonRs expects a list with DICOM JSON as elements.
  # Add study level tags to the SR.
  dataset = metadata_json.copy()
  series_uid = GenerateUID()
  instance_uid = GenerateUID()
  dicom_json.Insert(dataset, tags.SOP_CLASS_UID, tag_values.BASIC_TEXT_SR_CUID)
  dicom_json.Insert(dataset, tags.MODALITY, tag_values.SR_MODALITY)
  dicom_json.Insert(dataset, tags.SERIES_INSTANCE_UID, series_uid)
  dicom_json.Insert(dataset, tags.SPECIFIC_CHARACTER_SET, _ISO_CHARACTER_SET)

  logging.log(
      logging.INFO,
      'Creating DICOM JSON SR with Series UID: %s and Instance UID: %s',
      series_uid, instance_uid)
  dicom_json.Insert(dataset, tags.SOP_INSTANCE_UID, instance_uid)

  content_dataset = {}
  dicom_json.Insert(content_dataset, tags.RELATIONSHIP_TYPE, 'CONTAINS')
  dicom_json.Insert(content_dataset, tags.VALUE_TYPE, 'TEXT')
  dicom_json.Insert(content_dataset, tags.TEXT_VALUE, report_text)
  dicom_json.Insert(dataset, tags.CONTENT_SEQUENCE, content_dataset)

  dicom_json.Insert(dataset, tags.TRANSFER_SYNTAX_UID,
                    _IMPLICIT_VR_LITTLE_ENDIAN)
  dicom_json.Insert(dataset, tags.MEDIA_STORAGE_SOP_CLASS_UID,
                    tag_values.BASIC_TEXT_SR_CUID)
  dicom_json.Insert(dataset, tags.MEDIA_STORAGE_SOP_INSTANCE_UID, instance_uid)

  return dicom_json.ObjectWithBulkData(dataset)


def BuildJsonSC(image_array: np.ndarray, metadata_json: Dict[Text, Any],
                series_uid: Text) -> dicom_json.ObjectWithBulkData:
  """Builds and returns a DICOM Secondary Capture.

  Args:
    image_array: Image array (RGB) to embed in DICOM instance.
    metadata_json: Dict of tags (including study-level information) to add.
    series_uid: UID of the series to create the SC in.

  Returns:
    DICOM JSON Object containing JSON and bulk data of the Secondary Capture.
  """
  # Copy over any study and instance level tags.
  instance_uid = GenerateUID()
  metadata_json = metadata_json.copy()
  dicom_json.Insert(metadata_json, tags.SOP_CLASS_UID,
                    tag_values.SECONDARY_CAPTURE_CUID)
  dicom_json.Insert(metadata_json, tags.MODALITY, tag_values.OT_MODALITY)
  dicom_json.Insert(metadata_json, tags.SERIES_INSTANCE_UID, series_uid)
  dicom_json.Insert(metadata_json, tags.SPECIFIC_CHARACTER_SET,
                    _ISO_CHARACTER_SET)
  dicom_json.Insert(metadata_json, tags.SOP_INSTANCE_UID, instance_uid)
  dicom_json.Insert(metadata_json, tags.TRANSFER_SYNTAX_UID,
                    _IMPLICIT_VR_LITTLE_ENDIAN)
  dicom_json.Insert(metadata_json, tags.MEDIA_STORAGE_SOP_CLASS_UID,
                    tag_values.SECONDARY_CAPTURE_CUID)
  dicom_json.Insert(metadata_json, tags.MEDIA_STORAGE_SOP_INSTANCE_UID,
                    instance_uid)
  # Assures URI is unique.
  study_uid = dicom_json.GetValue(metadata_json, tags.STUDY_INSTANCE_UID)
  uri = '{}/{}/{}'.format(study_uid, series_uid, instance_uid)
  metadata_json[tags.PIXEL_DATA.number] = {
      'vr': tags.PIXEL_DATA.vr,
      'BulkDataURI': uri
  }

  dicom_json.Insert(metadata_json, tags.PHOTOMETRIC_INTERPRETATION, 'RGB')
  dicom_json.Insert(metadata_json, tags.SAMPLES_PER_PIXEL, 3)
  # Indicates we store pixel data as R1,G1,B1,R2,G2,B2...
  dicom_json.Insert(metadata_json, tags.PLANAR_CONFIGURATION, 0)
  dicom_json.Insert(metadata_json, tags.ROWS, image_array.shape[0])
  dicom_json.Insert(metadata_json, tags.COLUMNS, image_array.shape[1])
  dicom_json.Insert(metadata_json, tags.BITS_ALLOCATED, 8)
  dicom_json.Insert(metadata_json, tags.BITS_STORED, 8)
  dicom_json.Insert(metadata_json, tags.HIGH_BIT, 7)
  dicom_json.Insert(metadata_json, tags.PIXEL_REPRESENTATION, 0)

  bulkdata = dicom_web.DicomBulkData(
      uri=uri,
      data=image_array.tobytes(),
      content_type='application/octet-stream')
  return dicom_json.ObjectWithBulkData(metadata_json, [bulkdata])


def BuildJsonInstanceFromPng(
    image: bytes, sop_class_uid: Text) -> dicom_json.ObjectWithBulkData:
  """Builds and returns a DICOM instance from a PNG.

  This function will create a new DICOM study and series. Converts all incoming
  images to grayscale.

  Args:
    image: Image bytes of DICOM instance.
    sop_class_uid: UID of the SOP class for DICOM instance.

  Returns:
    DICOM JSON Object containing JSON and bulk data of the Secondary Capture.
  """
  study_uid = GenerateUID()
  series_uid = GenerateUID()
  instance_uid = GenerateUID()
  metadata_json = {}
  dicom_json.Insert(metadata_json, tags.PLANAR_CONFIGURATION, 0)
  # Converts colored images to grayscale.
  dicom_json.Insert(metadata_json, tags.PHOTOMETRIC_INTERPRETATION,
                    'MONOCHROME2')
  dicom_json.Insert(metadata_json, tags.SOP_CLASS_UID, sop_class_uid)
  dicom_json.Insert(metadata_json, tags.STUDY_INSTANCE_UID, study_uid)
  dicom_json.Insert(metadata_json, tags.SERIES_INSTANCE_UID, series_uid)
  dicom_json.Insert(metadata_json, tags.SPECIFIC_CHARACTER_SET,
                    _ISO_CHARACTER_SET)
  dicom_json.Insert(metadata_json, tags.SOP_INSTANCE_UID, instance_uid)
  dicom_json.Insert(metadata_json, tags.TRANSFER_SYNTAX_UID,
                    _EXPLICIT_VR_LITTLE_ENDIAN)
  dicom_json.Insert(metadata_json, tags.MEDIA_STORAGE_SOP_CLASS_UID,
                    sop_class_uid)
  dicom_json.Insert(metadata_json, tags.MEDIA_STORAGE_SOP_INSTANCE_UID,
                    instance_uid)

  # Assures URI is unique.
  uri = '{}/{}/{}'.format(study_uid, series_uid, instance_uid)
  metadata_json[tags.PIXEL_DATA.number] = {
      'vr': tags.PIXEL_DATA.vr,
      'BulkDataURI': uri
  }

  bulkdata = dicom_web.DicomBulkData(
      uri=uri, data=image, content_type='image/png; transfer-syntax=""')
  return dicom_json.ObjectWithBulkData(metadata_json, [bulkdata])


def GenerateUID(prefix: Text = _UUID_INFERENCE_PREFIX) -> Text:
  """Generates a random Instance UID in the correct format.

  Args:
    prefix: Text string that is the UUID prefix for the UUID.

  Returns:
    Unique UID with the provided prefix.
  """
  # Generates a unique UID using the Process ID, Host ID and current time.
  # Uses as a period as the separator and combines the generated UUID with the
  # provided prefix.
  # Example: 1.3.6.1.4.1.11129.5.3.268914880332007.160162.47.376673
  uuid_components = [
      prefix,
      uuid.getnode(),
      abs(os.getpid()),
      datetime.datetime.today().second,
      datetime.datetime.today().microsecond
  ]
  generated_uuid = '.'.join(
      str(uuid_component) for uuid_component in uuid_components)
  return generated_uuid


def IsInferenceToolkitUUID(uid: Text) -> bool:
  """Determines if the uid was generated by the inference module toolkit.

  Args:
    uid: Text string representing a UID.

  Returns:
    bool determing whether the UID has the GoogleInferenceUUID prefix.
  """
  return uid.find(_UUID_INFERENCE_PREFIX + '.') == 0
