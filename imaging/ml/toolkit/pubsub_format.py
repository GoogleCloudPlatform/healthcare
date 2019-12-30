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
from __future__ import google_type_annotations
from __future__ import print_function

import enum
import posixpath
from typing import Any, Text, Optional

from absl import logging
import attr
import six

from google.cloud import pubsub_v1
from google.rpc import code_pb2
from toolkit import dicom_web
from toolkit import exception
from util.regexp.re2.python import re2

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


class PathType(enum.Enum):
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
  dicomweb_url = attr.ib()  # type: str
  study_uid = attr.ib()  # type: str
  study_path = attr.ib()  # type: str
  series_uid = attr.ib(default=None)  # type: Optional[str]
  series_path = attr.ib(default=None)  # type: Optional[str]
  instance_uid = attr.ib(default=None)  # type: Optional[str]
  instance_path = attr.ib(default=None)  # type: Optional[str]
  conflict = attr.ib(default=DEFAULT_CONFLICT)  # type: ConflictType
  verification_test = attr.ib(default=False)  # type: bool
  prior_study_uid = attr.ib(default=None)  # type: Optional[str]
  prior_series_uid = attr.ib(default=None)  # type: Optional[str]
  output_dicom_store_path = attr.ib(default=None)  # type: Optional[str]


def ParseMessage(message: pubsub_v1.types.PubsubMessage,
                 path_type: PathType) -> ParsedMessage:
  """Parses input Pub/Sub message into a ParsedMessage object.

  Args:
    message: Pub/Sub message to be parsed. Expected to contain a path to a
      Study, Series, or Instance.
    path_type: indicates what the message is about (i.e. study, series or
      instance).

  Returns:
    ParsedMessage: Named tuple representing parsed Pub/Sub message data.

  Raises:
    exception.CustomExceptionError with status code INVALID_ARGUMENT if the
      input doesn't match expected format.
  """
  if path_type == PathType.DICOM_STORE:
    raise exception.CustomExceptionError(
        'DICOM_STORE path type not supported for input Pub/Sub messages.',
        code_pb2.Code.INVALID_ARGUMENT)
  # In PY2 bytes do not need to be decoded because they are equivalent to
  # strings.
  if six.PY2:
    input_path = message.data
  else:
    input_path = message.data.decode('utf-8')
  verification_test = (
      message.attributes.get('verification_test') in ['True', 'true'])
  conflict_attr = message.attributes.get('conflict')
  if conflict_attr and conflict_attr not in CONFLICT_MAP:
    raise exception.CustomExceptionError(
        'Unexpected value for conflict attribute: %s. Must be one of the'
        'following values: %s' % (conflict_attr, CONFLICT_MAP.keys()),
        code_pb2.Code.INVALID_ARGUMENT)
  conflict = CONFLICT_MAP[conflict_attr] if conflict_attr else DEFAULT_CONFLICT

  input_match = _ValidatePath(input_path, path_type)
  project_id = input_match.group(1)
  location_id = input_match.group(2)
  dataset_id = input_match.group(3)
  dicom_store_id = input_match.group(4)
  study_uid = input_match.group(5)
  study_path = posixpath.join('projects', project_id, 'locations', location_id,
                              'datasets', dataset_id, 'dicomStores',
                              dicom_store_id, 'dicomWeb', 'studies', study_uid)
  dicomweb_url = posixpath.join(dicom_web.CLOUD_HEALTHCARE_API_URL, 'projects',
                                project_id, 'locations', location_id,
                                'datasets', dataset_id, 'dicomStores',
                                dicom_store_id, 'dicomWeb')
  # Build the basic message.
  parsed_message = ParsedMessage(
      study_uid=study_uid,
      study_path=study_path,
      dicomweb_url=dicomweb_url,
      conflict=conflict,
      verification_test=verification_test)

  # Set the output DICOM store path, if available.
  output_dicom_store_path = message.attributes.get('output_dicom_store_path')
  if output_dicom_store_path is not None:
    # Pub/Sub message attributes are of type unicode in PY2 and must be
    # encoded to strings.
    if six.PY2:
      output_dicom_store_path = output_dicom_store_path.encode('utf-8')

    _ValidatePath(output_dicom_store_path, PathType.DICOM_STORE)
    parsed_message.output_dicom_store_path = output_dicom_store_path

  # Fill out the rest of the message, conditionally.
  if path_type in (PathType.SERIES, PathType.INSTANCE):
    series_uid = input_match.group(6)
    parsed_message.series_uid = series_uid
    parsed_message.series_path = posixpath.join('projects', project_id,
                                                'locations', location_id,
                                                'datasets', dataset_id,
                                                'dicomStores', dicom_store_id,
                                                'dicomWeb', 'studies',
                                                study_uid, 'series', series_uid)
    if path_type is PathType.INSTANCE:
      instance_uid = input_match.group(7)
      parsed_message.instance_uid = instance_uid
      parsed_message.instance_path = posixpath.join(
          'projects', project_id, 'locations', location_id, 'datasets',
          dataset_id, 'dicomStores', dicom_store_id, 'dicomWeb', 'studies',
          study_uid, 'series', series_uid, 'instances', instance_uid)

  # This could be expanded to accommodate other resource types as priors, but
  # currently it's just series.
  prior_series_path = message.attributes.get('prior_series_path')
  if prior_series_path:
    if prior_series_path in ['None', 'none']:
      parsed_message.prior_study_uid = NO_PRIOR
    else:
      prior_series_match = _ValidatePath(prior_series_path, PathType.SERIES)
      parsed_message.prior_study_uid = prior_series_match.group(5)
      parsed_message.prior_series_uid = prior_series_match.group(6)
  return parsed_message


def _ValidatePath(input_path: Text, path_type: PathType) -> Any:
  """Validates input_path to be in the expected format.

  Args:
    input_path: Path to an Instance, Series or Study in text format.
    path_type: The expected type of the path.

  Returns:
    Regex match object.

  Raises:
      exception.CustomExceptionError with type MALFORMED_INPUT_ERROR if the
        input_path is not of the expected format.
  """
  dicom_store_path_regex = (r'projects/([^/]+)/locations/([^/]+)/datasets/'
                            '([^/]+)/dicomStores/([^/]+)')
  study_path_regex = dicom_store_path_regex + '/dicomWeb/studies/([^/]+)'
  if path_type is PathType.DICOM_STORE:
    regexp = dicom_store_path_regex + (r'/?$')
  elif path_type is PathType.STUDY:
    regexp = study_path_regex + (r'/?$')
  elif path_type is PathType.SERIES:
    regexp = study_path_regex + (r'/series/([^/]+)/?$')
  elif path_type is PathType.INSTANCE:
    regexp = study_path_regex + (r'/series/([^/]+)/instances/([^/]+)/?$')

  match = re2.match(regexp, input_path)

  if ((match is None) or
      (path_type is PathType.DICOM_STORE and len(match.groups()) != 4) or
      (path_type is PathType.STUDY and len(match.groups()) != 5) or
      (path_type is PathType.SERIES and len(match.groups()) != 6) or
      (path_type is PathType.INSTANCE and len(match.groups()) != 7)):
    logging.log(logging.ERROR, 'Unexpected input path format: %s', input_path)
    raise exception.CustomExceptionError(
        'Unexpected input path format: %s' % input_path,
        code_pb2.Code.INVALID_ARGUMENT)
  return match
