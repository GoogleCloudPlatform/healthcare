#!/usr/bin/python3
#
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

r"""Cloud Functions implementation which takes a patient bundle from a FHIR
Store whenever a questionnaire gets answered, runs prediction against a
pre-trained model and writes the results back to the same FHIR Store.
"""

import base64
import datetime
import googleapiclient.discovery
import google.auth
import json
import logging
import os

from google.auth.transport.urllib3 import AuthorizedHttp
from utils import *

# These should be passed in through deployment.
MODEL = os.environ.get('MODEL')
VERSION = os.environ.get('VERSION')

FHIR_STORE_ENDPOINT_PREFIX = 'https://healthcare.googleapis.com/v1beta1'
CREATE_RESOURCE_ACTION = 'CreateResource'
UPDATE_RESOURCE_ACTION = 'UpdateResource'

RISKS = ['negligible', 'low', 'moderate', 'high', 'certain']

LOGGER = logging.getLogger('main')

def get_resource(http, resource_name):
  """Fetches a resource from the FHIR Store.

    Args:
      resource_name (str): the name of the resource, e.g. 'projects/my-project
        /locations/us-central1/datasets/my-dataset/fhirStores/my-store
        /fhir/Patient/patient-id'
    Returns:
      Object: the resource loaded from the FHIR Store.
  """
  response = http.request('GET', format_url(resource_name))
  if response.status > 299:
    LOGGER.critical("Failed to retrieve resource %s, response: %s" % (
      resource_name, response.data))
    return None

  return json.loads(response.data)


def build_risk_assessment(pid, qid, disease, risk, rid=None):
  """Builds a risk assessment JSON object.

    Returns:
      Str: JSON representation of a RiskAssessment resource.
  """
  risk_assessment = {
    'resourceType': RISKASSESSMENT_TYPE,
    'basis': [{'reference': pid}, {'reference': qid}],
    'status': 'final',
    'subject': {'reference': pid},
    'occurrenceDateTime':
      datetime.datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ'),
    'prediction': [{
      'outcome': {
        'coding': [{'display': disease}],
      },
      'qualitativeRisk': {
        'coding': [{
          'system': "http://hl7.org/fhir/risk-probability",
          'code': risk
        }]
      }
    }]
  }

  if rid is not None:
    risk_assessment['id'] = rid

  return json.dumps(risk_assessment)


def get_action(data):
  """Reads operation action (e.g. Create or Update) from pubsub message."""
  if data['attributes'] is not None:
    return data['attributes']['action']
  return None


def format_url(path, query=None):
  """Formats request URL with path and query string."""
  if query is None:
    return "%s/%s" % (FHIR_STORE_ENDPOINT_PREFIX, path)
  else:
    return "%s/%s?%s" % (FHIR_STORE_ENDPOINT_PREFIX, path, query)


def create_or_update_resource(http, path, payload):
  """Writes a resource to the FHIR Store.

    Args:
      path (str): path to the endpoint, e.g. 'projects/my-project
        /locations/us-central1/datasets/my-dataset/fhirStores/my-store
        /fhir/Patient' for create requests and 'projects/my-project
        /locations/us-central1/datasets/my-dataset/fhirStores/my-store
        /fhir/Patient/patient-id' for update requests.
      payload (str): resource to be written to the FHIR Store.
    Returns:
      Object: the resource from the server, usually this is an
        OperationOutcome resource if there is anything wrong.
  """
  # Determine which HTTP method we need to use: POST for create, and PUT for
  # update. The path of update requests have one more component than create
  # requests.
  method = 'POST' if path.count('/') == 9 else 'PUT'
  response = http.request(method, format_url(path), body=payload,
    headers={'Content-Type': 'application/fhir+json;charset=utf-8'})
  if response.status > 299:
    LOGGER.error("Failed to create or update resource %s, response: %s" % (
      payload, response.data))
    return None

  return json.loads(response.data)


def search_resource(http, path, query):
  """Searches a resource in the FHIR Store.

    Args:
      path (str): path to the search endpoint, e.g. 'projects/my-project
        /locations/us-central1/datasets/my-dataset/fhirStores/my-store
        /fhir/Patient'
      query (str): query parameter, e.g. 'age=gt30'
    Returns:
      List[dict]: a list of resources matching the search criteria.
  """
  response = http.request('GET', format_url(path, query=query))
  if response.status > 299:
    LOGGER.error("Failed to search resource %s, response: %s" % (query,
      response.data))
    return None

  bundle = json.loads(response.data)
  return list(map(lambda r: r['resource'], bundle['entry']))


def filter_resource(resources, qid, disease):
  """Finds a RiskAssessment.

  The target references a certain QuestionnaireResponse and is about the
  specified disease
  """
  def match(res):
    return extract_qid(res) == qid and extract_disease(res) == disease

  return next(filter(match, resources), None)


def build_examples(patient, questionnaire_response):
  """Builds examples to be sent for prediction.

  Two examples are created for the two diseases we are targeting at.
  """

  def map_example(disease):
    return {
      'age': calculate_age(patient['birthDate']),
      'gender': 1 if patient['gender'] == 'male' else 0,
      'country': COUNTRY_MAP[extract_country(questionnaire_response)],
      'duration': calculate_duration(
        *extract_start_end_date(questionnaire_response)),
      'disease': disease
    }

  return list(map(map_example, range(len(DISEASE_MAP))))


def predict(examples):
  """Sends features to Cloud ML Engine for online prediction.

    Args:
      examples (list): features to be fed into the model for prediction.
    Returns:
      Mapping[str: any]: dictionary of prediction results defined by the model.
  """
  service = googleapiclient.discovery.build('ml', 'v1', cache_discovery=False)
  name = "projects/%s/models/%s/versions/%s" % (
    os.environ.get('GCP_PROJECT'), MODEL, VERSION)

  response = service.projects().predict(name=name,
    body={'instances': examples}).execute()

  if 'error' in response:
    LOGGER.error("Prediction failed: %s" % response['error'])
    return None

  return response['predictions']


def main(data, context):
  """Extracts features from a patient bundle for online prediction.

  This process is broken down into a few steps:

  1. Fetch the QuestionnaireResponse we get triggered on (note that we
     only react to this resource type), and extract the patient that
     answered it.
  2. Fetch everything for the patient from step 1, and extract the
     features we are interested in.
  3. Send the features to Cloud ML for online prediction, and write the
     results back to the FHIR store.

  Args:
    data (dict): Cloud PubSub payload. The `data` field is what we are
    looking for.
    context (google.cloud.functions.Context): Metadata for the event.
  """

  if 'data' not in data:
    LOGGER.info('`data` field is not present, skipping...')
    return

  resource_name = base64.b64decode(data['data']).decode('utf-8')
  if QUESTIONNAIRERESPONSE_TYPE not in resource_name:
    LOGGER.info("Skipping resource %s which is irrelevant for prediction." %
      resource_name)
    return

  credentials, _ = google.auth.default()
  http = AuthorizedHttp(credentials)
  questionnaire_response = get_resource(http, resource_name)
  if questionnaire_response is None:
    return

  patient_id = questionnaire_response['subject']['reference']
  project_id, location, dataset_id, fhir_store_id, _ = _parse_resource_name(
    resource_name)
  patient = get_resource(http, _construct_resource_name(project_id, location,
    dataset_id, fhir_store_id, patient_id))
  if patient is None:
    return

  predictions = predict(build_examples(patient, questionnaire_response))
  if predictions is None:
    return

  pid = "%s/%s" % (PATIENT_TYPE, patient['id'])
  qid = "%s/%s" % (QUESTIONNAIRERESPONSE_TYPE, questionnaire_response['id'])

  action = get_action(data)
  for disease, idx in DISEASE_MAP.items():
    scores = predictions[idx]['probabilities']
    LOGGER.info("Prediction results: %s", scores)
    # Last element represents risk.
    score = scores[1]
    risk = RISKS[-1] if score == 1 else RISKS[int(score / 0.2)]

    path = _construct_resource_name(project_id, location, dataset_id,
      fhir_store_id, RISKASSESSMENT_TYPE)
    if action == UPDATE_RESOURCE_ACTION:
      resources = search_resource(http, path, "subject=%s" % pid)
      res = filter_resource(resources, qid, disease)
      if res is None:
        LOGGER.info("No existing RiskAssessment, createing a new one...")
        create_or_update_resource(http, path, build_risk_assessment(pid,
          qid, disease, risk))
        continue
      rid = res['id']

      path = _construct_resource_name(project_id, location, dataset_id,
        fhir_store_id, "%s/%s" % (RISKASSESSMENT_TYPE, rid))
      create_or_update_resource(http, path, build_risk_assessment(pid,
        qid, disease, risk, rid=rid))
    elif action == CREATE_RESOURCE_ACTION:
      create_or_update_resource(http, path, build_risk_assessment(pid,
        qid, disease, risk))


def _parse_resource_name(name):
  """Extracts project id, location, dataset id etc from the resource name."""
  parts = name.split('/')
  return parts[1], parts[3], parts[5], parts[7], \
      "%s/%s" % (parts[9], parts[10])


def _construct_resource_name(project_id, location, dataset_id, fhir_store_id,
  resource_id):
  """Constructs a resource name."""
  return '/'.join([
      'projects', project_id, 'locations', location, 'datasets', dataset_id,
      'fhirStores', fhir_store_id, 'fhir', resource_id
  ])
