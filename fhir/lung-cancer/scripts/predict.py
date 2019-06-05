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
r"""Verifies online prediction works.

Example usage:
python3 -m predict --project $PROJECT_ID --model $MODEL --version $VERSION
"""

import json
import os.path

from absl import app
from absl import flags

import googleapiclient.discovery

FLAGS = flags.FLAGS
flags.DEFINE_string('project', None, 'GCP project')
flags.DEFINE_string('model', None, 'The name of the deployed model.')
flags.DEFINE_string('version', None, 'The version of the deployed model.')
flags.mark_flag_as_required('project')
flags.mark_flag_as_required('model')


def predict(project, model, instances, version=None):
  """Send data to a deployed model for prediction.

  Args:
    project (str): project where the Cloud ML Engine Model is deployed.
    model (str): model name.
    instances (List[Mapping[str, Any]]): Keys should be the names of Tensors
      your deployed model expects as inputs. Values should be datatypes
      convertible to Tensors, or (potentially nested) lists of datatypes
      convertible to tensors.
    version (str): version of the model to target.

  Returns:
    Mapping[str: any]: dictionary of prediction results defined by the
      model.
  Raises:
    RuntimeError: an error occurred during prediction.
  """
  # Create the ML Engine service object.
  service = googleapiclient.discovery.build('ml', 'v1', cache_discovery=False)
  name = 'projects/{}/models/{}'.format(project, model)

  if version is not None:
    name += '/versions/' + version

  response = service.projects().predict(name=name, body=instances).execute()

  if 'error' in response:
    raise RuntimeError(response['error'])

  return response['predictions']


def main(_):
  path = os.path.join(
      os.path.abspath(os.path.dirname(__file__)), 'data/test.json')
  with open(path) as f:
    examples = json.load(f)
    results = predict(FLAGS.project, FLAGS.model, examples, FLAGS.version)
    for i, example in enumerate(examples['instances']):
      print('Input:', example)
      print('Risk: {:.2f}%'.format(results[i]['probabilities'][-1] * 100))


if __name__ == '__main__':
  app.run(main)
