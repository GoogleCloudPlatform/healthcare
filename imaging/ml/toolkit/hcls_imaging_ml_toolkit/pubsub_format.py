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
"""Pub/Sub message formatter and parser."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import enum
import traceback
from typing import Text, Optional

import attr

from google.cloud import pubsub_v1
from google.rpc import code_pb2
from hcls_imaging_ml_toolkit import dicom_path
from hcls_imaging_ml_toolkit import dicom_web
from hcls_imaging_ml_toolkit import exception

# Specifies that no prior should be used for running the inference flow for the
# input message.
NO_PRIOR = 'no_prior'


# Possible values for the "conflict" attribute of the Pub/Sub notification.
# The attribute controls how conflicts with existing DICOM instances containing
# inference module predictions are resolved.
class ConflictType(enum.Enum):
  # Raises an exception with status_code PREDICTION_EXISTS_ERROR.
  ABORT_CONFLICT = 1

  # Ignores conflicts and proceeds without change.
  IGNORE_CONFLICT = 2

  # Deletes all existing DICOM instances containing an inference module
  # prediction and then proceeds to create a new prediction.
  OVERWRITE_CONFLICT = 3


# TODO: Remove once switchover to dicom_path.Path is complete.
class PathTypeLegacy(enum.Enum):
  """Possible types of paths to DICOMweb resources."""

  # Indicates a path to a Study. Expected to be in the following format:
  # projects/<project_name>/locations/<location>/datasets/<dataset_name>
  # /dicomStores/<dicom_store_name>/dicomWeb/studies/<study_uid>
  STUDY = 1

  # Indicates a path to a Series. Expected to be in the following format:
  # projects/<project_name>/locations/<location>/datasets/<dataset_name>
  # /dicomStores/<dicom_store_name>/dicomWeb/studies/<study_uid>/series/
  # <series_uid>
  SERIES = 2

  # Indicates a path to an Instance. Expected to be in the following format:
  # projects/<project_name>/locations/<location>/datasets/<dataset_name>
  # /dicomStores/<dicom_store_name>/dicomWeb/studies/<study_uid>/series/
  # <series_uid>/instances/<instance_uid>
  INSTANCE = 3

  # Indicates a path to a DICOM Store. Expected to be in the following format:
  # projects/<project_name>/locations/<location>/datasets/<dataset_name>
  # /dicomStores/<dicom_store_name>
  DICOM_STORE = 4


CONFLICT_MAP = {
    'abort': ConflictType.ABORT_CONFLICT,
    'ignore': ConflictType.IGNORE_CONFLICT,
    'overwrite': ConflictType.OVERWRITE_CONFLICT
}

DEFAULT_CONFLICT = ConflictType.ABORT_CONFLICT


@attr.s
class ParsedMessage(object):
  """ParsedMessage represents the parsed Pub/Sub message.

  Attributes:
    input_path: Path object referencing the DICOM resource specified in the
      message.
    conflict: Pub/Sub message conflict attribute.
    verification_test: If True, the module will capture intermediate results and
      save them in a StructuredReport.
    prior_study_uid: Prior Study UID. This comes from parsing the optional
      Pub/Sub message |prior_series_path| attribute. This allows for explicit
      prior selection. If not set regular prior search flow will proceed.
    prior_series_uid: Prior Series UID. This comes from parsing the optional
      Pub/Sub message |prior_series_path| attribute.
    output_dicom_store_path: DICOMStore path to write inference results to. This
      comes from parsing the optional Pub/Sub message |output_dicom_store_path|
      attribute. This allows for results to be written to a different
      DICOMStore.
  """
  input_path = attr.ib(type=dicom_path.Path)
  conflict = attr.ib(type=ConflictType, default=DEFAULT_CONFLICT)
  verification_test = attr.ib(type=bool, default=False)
  prior_study_uid = attr.ib(type=Optional[Text], default=None)
  prior_series_uid = attr.ib(type=Optional[Text], default=None)
  output_dicom_store_path = attr.ib(
      type=Optional[dicom_path.Path], default=None)


def ParseMessage(message: pubsub_v1.types.PubsubMessage,
                 path_type: dicom_path.Type) -> ParsedMessage:
  """Parses input Pub/Sub message into a ParsedMessage object.

  Args:
    message: Pub/Sub message to be parsed. Expected to contain a DICOMweb path
      starting from "projects/" and down to the level of a Study UID,
      Series UID, or Instance UID.
    path_type: indicates the expected type of the DICOM resource the path in the
      message points to.

  Returns:
    ParsedMessage object representing parsed Pub/Sub message data.

  Raises:
    exception.CustomExceptionError with status code INVALID_ARGUMENT if the
      input doesn't match expected format.
  """
  input_path_str = message.data.decode()
  # Support both 'True' and 'true' for user convenience with manual invocation.
  verification_test = (
      message.attributes.get('verification_test') in ['True', 'true'])
  conflict_attr = message.attributes.get('conflict')
  if conflict_attr and conflict_attr not in CONFLICT_MAP:
    raise exception.CustomExceptionError(
        'Unexpected value for conflict attribute: %s. Must be one of the '
        'following values: %s' % (conflict_attr, CONFLICT_MAP.keys()),
        code_pb2.Code.INVALID_ARGUMENT)
  conflict = CONFLICT_MAP[conflict_attr] if conflict_attr else DEFAULT_CONFLICT

  try:
    input_path = dicom_path.FromString(input_path_str, path_type)
    parsed_message = ParsedMessage(
        input_path=input_path,
        conflict=conflict,
        verification_test=verification_test)

    # Set the output DICOM store path, if available.
    output_store_path_str = message.attributes.get('output_dicom_store_path')
    if output_store_path_str is not None:
      output_store_path = dicom_path.FromString(output_store_path_str,
                                                dicom_path.Type.STORE)
      parsed_message.output_dicom_store_path = output_store_path

    # This could be expanded to accommodate other resource types as priors, but
    # currently it's just series.
    prior_series_path_str = message.attributes.get('prior_series_path')
    if prior_series_path_str:
      if prior_series_path_str in ['None', 'none']:
        parsed_message.prior_study_uid = NO_PRIOR
      else:
        prior_series_path = dicom_path.FromString(prior_series_path_str,
                                                  dicom_path.Type.SERIES)
        parsed_message.prior_study_uid = prior_series_path.study_uid
        parsed_message.prior_series_uid = prior_series_path.series_uid
    return parsed_message
  except ValueError as e:
    traceback.print_exc()
    raise exception.CustomExceptionError(str(e), code_pb2.Code.INVALID_ARGUMENT)


# TODO: Remove once switchover to dicom_path.Path is complete.
@attr.s
class ParsedMessageLegacy(object):
  """ParsedMessageLegacy represents the parsed Pub/Sub message.

  Note on paths: Path is the part of the URL after the API path, e.g., starting
  from "projects/..." and until the end of the URL.

  Attributes:
    dicomweb_url: The dicom URL.
    study_uid: Study UID.
    study_path: Path to study.
    series_uid: Series UID - optional.
    series_path: Path to series - optional.
    instance_uid: Instance UID - optional.
    instance_path: Path to instance - optional.
    conflict: Pub/Sub message conflict attribute.
    verification_test: If True, the module will capture intermediate results and
      save them in a StructuredReport.
    prior_study_uid: Prior Study UID. This comes from parsing the optional
      Pub/Sub message |prior_series_path| attribute. This allows for explicit
      prior selection. If not set regular prior search flow will proceed.
    prior_series_uid: Prior Series UID. This comes from parsing the optional
      Pub/Sub message |prior_series_path| attribute.
    output_dicom_store_path: DICOMStore path to write inference results to. This
      comes from parsing the optional Pub/Sub message |output_dicom_store_path|
      attribute. This allows for results to be written to a different
      DICOMStore.
  """
  dicomweb_url = attr.ib(type=Text)
  study_uid = attr.ib(type=Text)
  study_path = attr.ib(type=Text)
  series_uid = attr.ib(type=Optional[Text], default=None)
  series_path = attr.ib(type=Optional[Text], default=None)
  instance_uid = attr.ib(type=Optional[Text], default=None)
  instance_path = attr.ib(type=Optional[Text], default=None)
  conflict = attr.ib(type=ConflictType, default=DEFAULT_CONFLICT)
  verification_test = attr.ib(type=bool, default=False)
  prior_study_uid = attr.ib(type=Optional[Text], default=None)
  prior_series_uid = attr.ib(type=Optional[Text], default=None)
  output_dicom_store_path = attr.ib(type=Optional[Text], default=None)


# TODO: Remove once switchover to dicom_path.Path is complete.
def _FromLegacyPathType(path_type_legacy: PathTypeLegacy) -> dicom_path.Type:
  """Converts a legacy DICOM path type to the actual path type."""
  if path_type_legacy == PathTypeLegacy.DICOM_STORE:
    return dicom_path.Type.STORE
  elif path_type_legacy == PathTypeLegacy.STUDY:
    return dicom_path.Type.STUDY
  elif path_type_legacy == PathTypeLegacy.SERIES:
    return dicom_path.Type.SERIES
  return dicom_path.Type.INSTANCE


# TODO: Remove once switchover to dicom_path.Path is complete.
def ParseMessageLegacy(message: pubsub_v1.types.PubsubMessage,
                       path_type_legacy: PathTypeLegacy) -> ParsedMessageLegacy:
  """Parses input Pub/Sub message into a ParsedMessage object.

  Args:
    message: Pub/Sub message to be parsed. Expected to contain a path to a
      Study, Series, or Instance.
    path_type_legacy: indicates what the message is about (i.e. study, series or
      instance).

  Returns:
    ParsedMessage: Named tuple representing parsed Pub/Sub message data.

  Raises:
    exception.CustomExceptionError with status code INVALID_ARGUMENT if the
      input doesn't match expected format.
  """
  path_type = _FromLegacyPathType(path_type_legacy)
  parsed_message = ParseMessage(message, path_type)
  input_path = parsed_message.input_path
  series_uid = input_path.series_uid
  series_path_str = (str(input_path.GetSeriesPath()) if series_uid else None)
  instance_uid = input_path.instance_uid
  instance_path_str = str(input_path) if instance_uid else None
  output_path = parsed_message.output_dicom_store_path

  return ParsedMessageLegacy(
      dicomweb_url=dicom_web.PathStrToUrl(input_path.dicomweb_path_str),
      study_uid=input_path.study_uid,
      study_path=str(input_path.GetStudyPath()),
      series_uid=series_uid,
      series_path=series_path_str,
      instance_uid=instance_uid,
      instance_path=instance_path_str,
      conflict=parsed_message.conflict,
      verification_test=parsed_message.verification_test,
      prior_study_uid=parsed_message.prior_study_uid,
      prior_series_uid=parsed_message.prior_series_uid,
      output_dicom_store_path=str(output_path) if output_path else None)
