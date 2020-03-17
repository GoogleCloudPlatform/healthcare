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
"""Utilities for DICOMweb path manipulation."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import enum
import posixpath
import re
from typing import Match, Text, Optional

import attr


class Type(enum.Enum):
  """Type of a resource the path points to."""
  STORE = 'store'
  STUDY = 'study'
  SERIES = 'series'
  INSTANCE = 'instance'


_ATTR_VALIDATOR_ID_1 = attr.validators.matches_re(r'[\w-]+')
_ATTR_VALIDATOR_ID_2 = attr.validators.matches_re(r'[\w.-]+')
_ATTR_VALIDATOR_UID = attr.validators.optional(
    attr.validators.matches_re(r'[\d.]+'))


@attr.s(frozen=True)
class Path(object):
  """Represents a path to a DICOM Store or a DICOM resource in CHC API.

  Attributes:
    project_id: Project ID. Must be non-empty.
    location: Location. Must be non-empty.
    dataset_id: Dataset ID. Must be non-empty.
    store_id: DICOM Store ID. Must be non-empty.
    study_uid: DICOM Study UID. Optional.
    series_uid: DICOM Series UID. Optional.
    instance_uid: DICOM Instance UID. Optional.
  """
  project_id = attr.ib(type=Text, validator=_ATTR_VALIDATOR_ID_1)
  location = attr.ib(type=Text, validator=_ATTR_VALIDATOR_ID_1)
  dataset_id = attr.ib(type=Text, validator=_ATTR_VALIDATOR_ID_2)
  store_id = attr.ib(type=Text, validator=_ATTR_VALIDATOR_ID_2)
  study_uid = attr.ib(
      default=None, type=Optional[Text], validator=_ATTR_VALIDATOR_UID)
  series_uid = attr.ib(
      default=None, type=Optional[Text], validator=_ATTR_VALIDATOR_UID)
  instance_uid = attr.ib(
      default=None, type=Optional[Text], validator=_ATTR_VALIDATOR_UID)

  @study_uid.validator
  def _StudyUidMissing(self, _, value: Optional[Text]) -> None:
    if not value:
      if self.series_uid or self.instance_uid:
        raise ValueError('study_uid missing with non-empty series_uid or '
                         'instance_uid. series_uid: %s, instance_uid: %s' %
                         (self.series_uid, self.instance_uid))

  @series_uid.validator
  def _SeriesUidMissing(self, _, value: Optional[Text]) -> None:
    if not value:
      if self.instance_uid:
        raise ValueError('series_uid missing with non-empty instance_uid. '
                         'instance_uid: %s' % self.instance_uid)

  def __str__(self):
    """Returns the text representation of the path."""
    store_path_str = posixpath.join('projects', self.project_id, 'locations',
                                    self.location, 'datasets', self.dataset_id,
                                    'dicomStores', self.store_id)
    if self.study_uid is None:
      return store_path_str

    study_path_str = posixpath.join(store_path_str, 'dicomWeb/studies',
                                    self.study_uid)
    if self.series_uid is None:
      return study_path_str

    series_path_str = posixpath.join(study_path_str, 'series', self.series_uid)
    if self.instance_uid is None:
      return series_path_str

    return posixpath.join(series_path_str, 'instances', self.instance_uid)

  @property
  def type(self) -> Type:
    """Type of the DICOM resource corresponding to the path."""
    if not self.study_uid:
      return Type.STORE
    elif not self.series_uid:
      return Type.STUDY
    elif not self.instance_uid:
      return Type.SERIES
    return Type.INSTANCE

  @property
  def dicomweb_path_str(self) -> Text:
    """Path to the DICOMweb endpoint for the DICOM Store."""
    return posixpath.join(str(self.GetStorePath()), 'dicomWeb')

  def GetStorePath(self) -> 'Path':
    """Returns the sub-path for the DICOM Store within this path."""
    return Path(self.project_id, self.location, self.dataset_id, self.store_id)

  def GetStudyPath(self) -> 'Path':
    """Returns the sub-path for the DICOM Study within this path."""
    if self.type == Type.STORE:
      raise ValueError("Can't get a study path from a store path.")
    return Path(self.project_id, self.location, self.dataset_id, self.store_id,
                self.study_uid)

  def GetSeriesPath(self) -> 'Path':
    """Returns the sub-path for the DICOM Series within this path."""
    if self.type in (Type.STORE, Type.STUDY):
      raise ValueError("Can't get a series path from a %s path." % self.type)
    return Path(self.project_id, self.location, self.dataset_id, self.store_id,
                self.study_uid, self.series_uid)


def _MatchRegex(regex: Text, text_str: Text, error_str) -> Match[Text]:
  """Matches the regex and returns the match or raises ValueError if failed."""
  match = re.match(regex, text_str)
  if match is None:
    raise ValueError(error_str)
  return match


def _FromString(path_str: Text) -> Path:
  """Parses the string and returns the Path object or raises ValueError if failed."""
  store_regex = (r'projects/([\w-]+)/locations/([\w-]+)/datasets/'
                 r'([\w.-]+)/dicomStores/([\w.-]+)(.*)')
  match_err_str = 'Error parsing the path. Path: %s' % path_str
  store_match = _MatchRegex(store_regex, path_str, match_err_str)

  project_id = store_match.group(1)
  location = store_match.group(2)
  dataset_id = store_match.group(3)
  store_id = store_match.group(4)
  store_path_suffix = store_match.group(5)

  if not store_path_suffix or store_path_suffix == '/':
    return Path(project_id, location, dataset_id, store_id)

  studies_regex = r'/dicomWeb/studies/([\d.]+)(.*)'
  studies_match = _MatchRegex(studies_regex, store_path_suffix, match_err_str)
  study_uid = studies_match.group(1)
  study_path_suffix = studies_match.group(2)

  if not study_path_suffix or study_path_suffix == '/':
    return Path(project_id, location, dataset_id, store_id, study_uid)

  series_regex = r'/series/([\d.]+)(.*)'
  series_match = _MatchRegex(series_regex, study_path_suffix, match_err_str)
  series_uid = series_match.group(1)
  series_path_suffix = series_match.group(2)

  if not series_path_suffix or series_path_suffix == '/':
    return Path(project_id, location, dataset_id, store_id, study_uid,
                series_uid)

  instance_regex = r'/instances/([\d.]+)/?$'
  instance_match = _MatchRegex(instance_regex, series_path_suffix,
                               match_err_str)
  instance_uid = instance_match.group(1)

  return Path(project_id, location, dataset_id, store_id, study_uid, series_uid,
              instance_uid)


def FromString(path_str: Text, path_type: Optional[Type] = None) -> Path:
  """Parses the string and returns the Path object or raises ValueError if failed.

  Args:
    path_str: The string containing the path.
    path_type: The expected type of the path or None if no specific type is
      expected.

  Returns:
    The newly constructed Path object.
  Raises:
    ValueError if the path cannot be parsed or the actual path type doesn't
      match the specified expected type.
  """
  path = _FromString(path_str)

  # Validate that the path is of the right type of the type is specified.
  if path_type and path.type != path_type:
    raise ValueError(
        'Unexpected path type. Expected: %s, actual: %s. Path: %s' %
        (type, path.type, path_str))

  return path


def FromPath(base_path: Path,
             store_id: Text = None,
             study_uid: Text = None,
             series_uid: Text = None,
             instance_uid: Text = None) -> Path:
  """Creates a new Path object based on the provided one.

  Replaces the specified path components in the base path to create the new one.

  Args:
    base_path: The base path to use.
    store_id: The store ID to use in the new path or None if the store ID from
      the base path should be used.
    study_uid: The study UID to use in the new path or None if the study UID
      from the base path should be used.
    series_uid: The series UID to use in the new path or None if the series UID
      from the base path should be used.
    instance_uid: The instance UID to use in the new path or None if the
      instance UID from the base path should be used.

  Returns:
    The newly constructed Path object.
  Raises:
    ValueError if the new path is invalid (e.g. if the instance UID is
      specified, but the series UID is None).
  """
  store_id = store_id if store_id else base_path.store_id
  study_uid = study_uid if study_uid else base_path.study_uid
  series_uid = series_uid if series_uid else base_path.series_uid
  instance_uid = instance_uid if instance_uid else base_path.instance_uid
  return Path(base_path.project_id, base_path.location, base_path.dataset_id,
              store_id, study_uid, series_uid, instance_uid)
