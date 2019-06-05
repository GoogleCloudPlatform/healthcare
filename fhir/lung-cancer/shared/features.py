#!/usr/bin/python3
#
# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Utilities for working with and building features for the model."""

from shared import utils


def get_smoker_status(observation):
  """Does `observation` represent a suvery response indicating that the patient is or was a smoker."""
  try:
    for coding in observation['valueCodeableConcept']['coding']:
      if ('system' in coding and 'code' in coding and
          coding['system'] == utils.SNOMED_SYSTEM and
          (coding['code'] == '8517006' or coding['code'] == '449868002'
          )  # Former smoker or Every day smoker
         ):
        return True
    return False
  except KeyError:
    return False


def get_cancer_status(condition):
  """Does `condition` represent a cancer diagnosis."""
  try:
    for coding in condition['code']['coding']:
      if ('system' in coding and 'code' in coding and
          coding['system'] == utils.SNOMED_SYSTEM and
          (coding['code'] == '254637007' or coding['code'] == '424132000' or
           coding['code'] == '162573006')):
        return True
    return False
  except KeyError:
    return False


def get_weight(observation):
  """If `observation` is a weight measurement then the patient's weight is returned."""
  try:
    for coding in observation['code']['coding']:
      if ('system' in coding and 'code' in coding and
          coding['system'] == utils.LOINC_SYSTEM and
          coding['code'] == '29463-7'):
        return int(observation['valueQuantity']['value'])
    return None
  except KeyError:
    return None


def build_example(patient_bundle):
  """Builds examples to be sent for prediction.

  Args:
    patient_bundle (Object):

  Returns:
    Dict[string, int]: the features for a model.
  """

  is_smoker = False
  age = 0
  has_cancer = False
  weight = None
  if 'entry' not in patient_bundle:
    return None

  for entry in patient_bundle['entry']:
    if 'resource' not in entry:
      continue
    resource = entry['resource']
    if resource['resourceType'] == utils.OBSERVATION_TYPE:
      is_smoker = is_smoker or get_smoker_status(resource)
      weight = get_weight(resource) or weight
    elif resource['resourceType'] == utils.PATIENT_TYPE:
      age = utils.calculate_age(resource)
    elif resource['resourceType'] == utils.CONDITION_TYPE:
      has_cancer = has_cancer or get_cancer_status(resource)

  return {
      'age': age,
      'weight': weight or 0,
      'is_smoker': 1 if is_smoker else 0,
      'has_cancer': 1 if has_cancer else 0
  }
