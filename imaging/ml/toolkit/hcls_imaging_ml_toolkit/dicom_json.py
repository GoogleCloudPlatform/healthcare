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
"""Utilities for manipulating DICOM JSON."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from typing import Any, Dict, List, Optional, Text
import attr

from hcls_imaging_ml_toolkit import tags

# The key used for values in DICOM JSON.
_VALUE_KEY = 'Value'


def Insert(dicom_json: Dict[Text, Any], tag: tags.DicomTag, value: Any) -> None:
  """Inserts a Dicom Tag into passed DICOM JSON Dict.

  Args:
    dicom_json: DICOM JSON dict where the tag will be inserted. See:
      https://www.dicomstandard.org/dicomweb/dicom-json-format
    tag: A DICOM tag.
    value: Any type that will be inserted into dict as the value for the tag.
  """
  tag_value = value if isinstance(value, list) else [value]
  dicom_json[tag.number] = {'vr': tag.vr, _VALUE_KEY: tag_value}


def GetList(dicom_json: Dict[Text, Any],
            tag: tags.DicomTag) -> Optional[List[Any]]:
  """Returns the value list for the tag from the provided DICOM JSON.

  Args:
    dicom_json: Dictionary containing DICOM JSON. See:
      https://www.dicomstandard.org/dicomweb/dicom-json-format
    tag: The tag to return the list of values for.

  Returns:
    The value list corresponding to the tag or None if the tag or value list is
    not present in the dictionary.
  """
  if tag.number not in dicom_json:
    return None
  return dicom_json[tag.number].get(_VALUE_KEY)


def GetValue(dicom_json: Dict[Text, Any], tag: tags.DicomTag) -> Any:
  """Returns the first value for the tag from the provided DICOM JSON.

  Returns the first value from the value list corresponding to the provided tag.
  For many DICOM tags this is going to be the only value in the list. If no
  value list exists, returns None.

  Args:
    dicom_json: Dictionary containing DICOM JSON. See:
      https://www.dicomstandard.org/dicomweb/dicom-json-format
    tag: The tag to return the value for.

  Returns:
    The first value from the value list corresponding to the tag or None if:
    - The tag or value list is not present in the dictionary.
    - The value list is empty.
  """
  value_list = GetList(dicom_json, tag)
  return value_list[0] if value_list else None


@attr.s
class DicomBulkData(object):
  # URI for the bulkdata.
  uri = attr.ib(type=Text)
  # The payload.
  data = attr.ib(type=bytes)
  # Content type in the following format:
  # https://www.w3.org/Protocols/rfc1341/4_Content-Type.html.
  content_type = attr.ib(type=Text)


@attr.s
class ObjectWithBulkData(object):
  """DICOM JSON object with the optional bulk data."""
  dicom_dict = attr.ib(type=Dict[Text, Any])
  bulkdata_list = attr.ib(type=List[DicomBulkData], factory=list)

  @property
  def instance_uid(self) -> Text:
    """Returns the Instance UID of the DICOM Object based on the DICOM data."""
    return GetValue(self.dicom_dict, tags.SOP_INSTANCE_UID)

  @property
  def series_uid(self) -> Text:
    """Returns the Series UID of the DICOM Object based on the DICOM data."""
    return GetValue(self.dicom_dict, tags.SERIES_INSTANCE_UID)

  @property
  def study_uid(self) -> Text:
    """Returns the Study UID of the DICOM Object based on the DICOM data."""
    return GetValue(self.dicom_dict, tags.STUDY_INSTANCE_UID)
