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
r"""Lung cancer Cloud Function.

Cloud Functions implementation which takes a patient bundle from a FHIR
Store whenever a Patient, Observation or Condition is changed, runs prediction
against a pre-trained model and writes the results back to the same FHIR Store.

"""

import base64
import datetime
import json
import logging
import os

import googleapiclient.discovery
from shared import features
from shared import utils

import google.auth
from google.auth.transport.urllib3 import AuthorizedHttp

# These should be passed in through deployment.
MODEL = os.environ.get('MODEL')
VERSION = os.environ.get('VERSION')

FHIR_STORE_ENDPOINT_PREFIX = 'https://healthcare.googleapis.com/v1beta1'
CREATE_RESOURCE_ACTION = 'CreateResource'
UPDATE_RESOURCE_ACTION = 'UpdateResource'

RISKS = ['negligible', 'low', 'moderate', 'high']

LOGGER = logging.getLogger('main')


def get_resource(http, resource_name):
  """Fetches a resource from the FHIR Store.

  Args:
    http (google.auth.transport.urllib3.AuthorizedHttp):
    resource_name (str): the name of the resource, e.g. 'projects/my-project
      /locations/us-central1/datasets/my-dataset/fhirStores/my-store
      /fhir/Patient/patient-id'

  Returns:
    Object: the resource loaded from the FHIR Store.
  """
  response = http.request('GET', format_url(resource_name))
  if response.status > 299:
    LOGGER.critical('Failed to retrieve resource %s, response: %s',
                    resource_name, response.data)
    return None

  return json.loads(response.data)


def build_risk_assessment(pid, risk, rid=None):
  """Builds a risk assessment JSON object.

  Args:
    pid (str): the patient ID for this risk assessment.
    risk (float): the predicted risk of lung cancer.
    rid (str): a previous risk assessment's ID that this one will overwrite.

  Returns:
    str: JSON representation of a RiskAssessment resource.
  """
  risk_assessment = {
      'resourceType':
          utils.RISKASSESSMENT_TYPE,
      'basis': [{
          'reference': pid
      }],
      'status':
          'final',
      'subject': {
          'reference': pid
      },
      'occurrenceDateTime':
          datetime.datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ'),
      'prediction': [{
          'outcome': {
              'coding': [{
                  'system': utils.SNOMED_SYSTEM,
                  'code': '162573006',
                  'display': 'Suspected lung cancer (situation)',
              }],
              'text': 'Suspected lung cancer (situation)',
          },
          'qualitativeRisk': {
              'coding': [{
                  'system': 'http://hl7.org/fhir/risk-probability',
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
    return '{}/{}'.format(FHIR_STORE_ENDPOINT_PREFIX, path)
  else:
    return '{}/{}?{}'.format(FHIR_STORE_ENDPOINT_PREFIX, path, query)


def create_or_update_resource(http, path, payload):
  """Writes a resource to the FHIR Store.

  Args:
    http (google.auth.transport.urllib3.AuthorizedHttp):
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
  response = http.request(
      method,
      format_url(path),
      body=payload,
      headers={'Content-Type': 'application/fhir+json;charset=utf-8'})
  if response.status > 299:
    LOGGER.error('Failed to create or update resource %s, response: %s',
                 payload, response.data)
    return None

  return json.loads(response.data)


def search_resource(http, path, query):
  """Searches a resource in the FHIR Store.

  Args:
    http (google.auth.transport.urllib3.AuthorizedHttp):
    path (str): path to the search endpoint, e.g. 'projects/my-project
      /locations/us-central1/datasets/my-dataset/fhirStores/my-store
      /fhir/Patient'
    query (str): query parameter, e.g. 'age=gt30'

  Returns:
    List[dict]: a list of resources matching the search criteria.
  """
  response = http.request('GET', format_url(path, query=query))
  if response.status > 299:
    LOGGER.error('Failed to search resource %s, response: %s', query,
                 response.data)
    return None

  bundle = json.loads(response.data)
  return [e['resource'] for e in bundle.get('entry', [])]


def filter_resource(resources):
  """Finds a RiskAssessment.

  A patient may have multiple risk assessments, so we filter them to find the
  one about this specified disease

  Args:
    resources (List[Object]): the lung cancer risk assessment or None.
  """
  return next(
      (res for res in resources
       if utils.extract_disease(res) == 'Suspected lung cancer (situation)'),
      None)


def predict(example):
  """Sends features to Cloud ML Engine for online prediction.

  Args:
    example (dict): features to be fed into the model for prediction.

  Returns:
    Mapping[str: any]: dictionary of prediction results defined by the model.
  """
  service = googleapiclient.discovery.build('ml', 'v1', cache_discovery=False)
  name = 'projects/{p}/models/{m}/versions/{v}'.format(
      p=os.environ.get('GCP_PROJECT'),
      m=MODEL,
      v=VERSION,
  )

  del example['has_cancer']
  response = service.projects().predict(
      name=name, body={
          'instances': [example],
      }).execute()

  if 'error' in response:
    LOGGER.error('Prediction failed: %s', response['error'])
    return None

  return response['predictions']


def get_patient_everything(http, patient_name):
  response = http.request(
      'GET', '{}/{}/$everything'.format(FHIR_STORE_ENDPOINT_PREFIX,
                                        patient_name))
  if response.status > 299:
    LOGGER.critical('Failed to retrieve resource %s, response: %s',
                    patient_name, response.data)
    return None

  return json.loads(response.data)


def create_or_update_risk_assessment(http, patient_name, predictions, action):
  """Creates or updates a risk assessment (if one already exists for the given patient)."""
  scores = predictions[0]['probabilities']
  LOGGER.info('Prediction results: %s', scores)
  # Last element represents risk.
  score = scores[1]
  risk = RISKS[min(int(score / 0.015), len(RISKS) - 1)]

  project_id, location, dataset_id, fhir_store_id, pid = _parse_resource_name(
      patient_name)
  path = _construct_resource_name(project_id, location, dataset_id,
                                  fhir_store_id, utils.RISKASSESSMENT_TYPE)

  if action == UPDATE_RESOURCE_ACTION:
    resources = search_resource(http, path, 'subject=' + pid)
    res = filter_resource(resources)
    if res is None:
      LOGGER.info('No existing RiskAssessment, creating a new one...')
      create_or_update_resource(http, path, build_risk_assessment(pid, risk))
      return

    rid = res['id']
    path = _construct_resource_name(
        project_id, location, dataset_id, fhir_store_id,
        '{}/{}'.format(utils.RISKASSESSMENT_TYPE, rid))
    create_or_update_resource(http, path,
                              build_risk_assessment(pid, risk, rid=rid))
  elif action == CREATE_RESOURCE_ACTION:
    create_or_update_resource(http, path, build_risk_assessment(pid, risk))


def get_corresponding_patient(http, resource_name, resource):
  """Gets the patient referenced by resource, or just returns resource if it's a patient."""
  if resource['resourceType'] == utils.PATIENT_TYPE:
    return resource
  if resource['resourceType'] == utils.CONDITION_TYPE:
    ref = resource['subject']['reference']
  elif resource['resourceType'] == utils.OBSERVATION_TYPE:
    ref = resource['subject']['reference']
  if utils.PATIENT_TYPE not in ref:
    return None
  project_id, location, dataset_id, fhir_store_id, _ = _parse_resource_name(
      resource_name)
  patient_name = _construct_resource_name(project_id, location, dataset_id,
                                          fhir_store_id, ref)
  return get_resource(http, patient_name)


# pylint: disable=unused-argument
def main(data, context):
  """Extracts features from a patient bundle for online prediction.

  This process is broken down into a few steps:

  1. Fetch the Resource we get triggered on, and fetch/extract the patient that
     it is related to.
  2. Fetch everything for the patient from step 1, and extract the
     features we are interested in.
  3. Send the features to Cloud ML for online prediction, and write the
     results back to the FHIR store.

  Args:
    data (dict): Cloud PubSub payload. The `data` field is what we are looking
      for.
    context (google.cloud.functions.Context): Metadata for the event.
  """

  if 'data' not in data:
    LOGGER.info('`data` field is not present, skipping...')
    return

  resource_name = base64.b64decode(data['data']).decode('utf-8')
  if (utils.CONDITION_TYPE not in resource_name and
      utils.PATIENT_TYPE not in resource_name and
      utils.OBSERVATION_TYPE not in resource_name):
    LOGGER.info('Skipping resource %s which is irrelevant for prediction.',
                resource_name)
    return

  credentials, _ = google.auth.default()
  http = AuthorizedHttp(credentials)
  resource = get_resource(http, resource_name)
  if resource is None:
    return

  patient = get_corresponding_patient(http, resource_name, resource)
  if patient is None:
    LOGGER.error('Could not find corresponding patient in resource %s',
                 resource_name)
    return

  project_id, location, dataset_id, fhir_store_id, _ = _parse_resource_name(
      resource_name)
  patient_id = 'Patient/{}'.format(patient['id'])
  patient_name = _construct_resource_name(project_id, location, dataset_id,
                                          fhir_store_id, patient_id)
  patient_bundle = get_patient_everything(http, patient_name)
  if patient_bundle is None:
    return

  predictions = predict(features.build_example(patient_bundle))
  if predictions is None:
    return

  action = get_action(data)
  create_or_update_risk_assessment(http, patient_name, predictions, action)


def _parse_resource_name(name):
  """Extracts project id, location, dataset id etc from the resource name."""
  parts = name.split('/')
  return parts[1], parts[3], parts[5], parts[7], '{}/{}'.format(
      parts[9], parts[10])


def _construct_resource_name(project_id, location, dataset_id, fhir_store_id,
                             resource_id):
  """Constructs a resource name."""
  return '/'.join([
      'projects', project_id, 'locations', location, 'datasets', dataset_id,
      'fhirStores', fhir_store_id, 'fhir', resource_id
  ])
