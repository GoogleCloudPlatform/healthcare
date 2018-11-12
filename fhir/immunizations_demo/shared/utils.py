# Copyright 2018 Google LLC
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
QUESTIONNAIRERESPONSE_TYPE = 'QuestionnaireResponse'
RISKASSESSMENT_TYPE = 'RiskAssessment'
CONDITION_TYPE = 'Condition'

COUNTRY_MAP = {
    'China': 0,
    'India': 1,
    'United States': 2,
    'Indonesia': 3,
    'Brazil': 4,
    'Pakistan': 5,
    'Bangladesh': 6,
    'Nigeria': 7,
    'Russia': 8,
    'Japan': 9,
    'Mexico': 10,
    'Philippines': 11,
    'Vietnam': 12,
    'Ethiopia': 13,
    'Germany': 14,
    'Egypt': 15,
    'Turkey': 16,
    'Iran': 17,
    'Democratic Republic of the Congo': 18,
    'Thailand': 19,
}
DISEASE_MAP = {
    'Hepatitis': 0,
    'Measles': 1,
    'Meningitis': 2,
    'Yellow Fever': 3,
}


def extract_uuid(res_id):
  """Extracts the UUID part of the resource id."""
  return res_id.split('/')[1]


def calculate_age(birth_date_str):
  """Calculates the age given brith date."""
  date = datetime.datetime.strptime(birth_date_str, "%Y-%m-%d").date()
  today = datetime.date.today()
  offset = 1 if date.month > today.month or (date.month == today.month
      and date.day > today.day) else 0
  return today.year - date.year - offset


def calculate_duration(start_date_str, end_date_str):
  """Calculates the duration of a stay."""
  start_date = datetime.datetime.strptime(start_date_str, "%Y-%m-%d").date()
  end_date = datetime.datetime.strptime(end_date_str, "%Y-%m-%d").date()
  return (end_date - start_date).days


def extract_country(questionnaire_response):
  """Extracts country information from a QuestionnaireResponse resource.

  Example resource:

  {
    ...
    "item": [
      {
        "answer": [{"valueString": "countryA"}],
        "linkId": "1"
      },
      {
        "item": [
          {
            "answer": [{"valueDate": "2018-10-15"}],
            "linkId": "2.1"
          },
          {
            "answer": [{"valueDate": "2018-10-22"}],
            "linkId": "2.2"
          }
        ],
        "linkId": "2"
      }
    ],
    "resourceType": "QuestionnaireResponse",
    ...
  }

  Right now the answers are fixed (i.e. first is country, then dates). So
  we hardcoded the indexes here and below.
  """
  return questionnaire_response['item'][0]['answer'][0]['valueString']


def extract_start_end_date(questionnaire_response):
  """Extracts start and end dates from a QuestionnaireResponse resource."""
  item = questionnaire_response['item'][1]['item']
  return item[0]['answer'][0]['valueDate'], item[1]['answer'][0]['valueDate']


def extract_disease(risk_assessment):
  """Extracts disease from a RiskAssessment resource.

  Example resource:
  {
    ...
    "basis": [
      {
        "reference": "Patient/610c6252-34de-4493-a3b4-f7ecc43e5681"
      },
      {
        "reference": "QuestionnaireResponse/9c318953-f6b3-492c-aa76-a9564a3cbeba"
      }
    ],
    "prediction": [
      {
        "outcome": {"coding": [{"display": "Measles"}]},
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
  """
  return risk_assessment['prediction'][0]['outcome']['coding'][0]['display']


def extract_risk(risk_assessment):
  """Extracts risk from a RiskAssessment resource."""
  prediction = risk_assessment['prediction']
  return prediction[0]['qualitativeRisk']['coding'][0]['code']


def extract_qid(risk_assessment):
  """Extracts the id of the QuestionnaireResponse the RiskAssessment links to.

  The generated data now always put the reference to the QuestionnaireResponse
  as the second item in the basis list.
  """
  return risk_assessment['basis'][1]['reference']


def extract_evidence_id(condition):
  """Extracts the id of the evidence resource linked to the Condition.

  In this demo, the evidence is always a QuestionnaireResponse, which is the
  only evidence and set on the detail field.

  Example resource:
  {
    ...
    "evidence":[
      {
        "detail":[
          {"reference":"QuestionnaireResponse/ab376d48-5fdc-4876-9441-c68cf59a8431"}
        ]
      }
    ],
    "resourceType":"Condition",
    ...
  }
  """
  return condition['evidence'][0]['detail'][0]['reference']


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
  """
  return condition['code']['coding'][0]['code']