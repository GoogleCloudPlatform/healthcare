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
r"""Utility variables and methods."""

import datetime

PATIENT_TYPE = 'Patient'
RISKASSESSMENT_TYPE = 'RiskAssessment'
CONDITION_TYPE = 'Condition'
OBSERVATION_TYPE = 'Observation'

SNOMED_SYSTEM = 'http://snomed.info/sct'
LOINC_SYSTEM = 'http://loinc.org'


def extract_uuid(res_id):
  """Extracts the UUID part of the resource id."""
  return res_id.split('/')[1]


def calculate_age(patient):
  """Calculates the age of a patient."""
  startdate = datetime.datetime.strptime(patient['birthDate'],
                                         '%Y-%m-%d').date()
  enddate = datetime.date.today()
  if 'deceasedDateTime' in patient:
    enddate = datetime.datetime.strptime(
        patient['deceasedDateTime'].split('T')[0], '%Y-%m-%d').date()
  return int((enddate - startdate).days / 365)


def extract_disease(risk_assessment):
  """Extracts disease from a RiskAssessment resource.

  Example resource:
  {
    ...
    "basis": [
      {
        "reference": "Patient/610c6252-34de-4493-a3b4-f7ecc43e5681"
      }
    ],
    "prediction": [
      {
        "outcome": {"coding": [{"display": "Lung cancer"}]},
        "qualitativeRisk": {
          "coding": [
            {
              "code": "low",
              "system": "http://hl7.org/fhir/risk-probability"
            }
          ]
        }
      }
    ],
    "resourceType": "RiskAssessment",
    ...
  }

  Design is that there is only one disease and risk probablity per resource, so
  we hardcoded the indexes here and below.

  Args:
    risk_assessment (Object):

  Returns:
    str: the disease the risk assessment predicts.
  """
  return risk_assessment['prediction'][0]['outcome']['coding'][0]['display']


def extract_risk(risk_assessment):
  """Extracts risk from a RiskAssessment resource."""
  prediction = risk_assessment['prediction']
  return prediction[0]['qualitativeRisk']['coding'][0]['code']


def extract_condition_disease(condition):
  """Extracts the disease encoded in the Condition resource.

  Example resource:
  {
    ...
    "code":{
      "coding":[
        {
          "code":"Yellow Fever",
          "system":"http://hl7.org/fhir/v3/ConditionCode"
        }
      ]
    }
    ...
  }

  Args:
    condition (Object):

  Returns:
    str: the disease in the condition.
  """
  return condition['code']['coding'][0]['code']
