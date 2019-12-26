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
r"""Loads resources exported from a FHIR Store to GCS (specifically in this
demo, Patient, QuestionnaireResponse and RiskAssessment) and extracts relevant
features and labels from the resources for training. The genereated tensorflow
record dataset is stored in a specified location on GCS.

Example usage:
python assemble_training_data.py --src_bucket=my-bucket \
                                 --src_folder=export    \
                                 --dst_bucket=my-bucket \
                                 --dst_folder=tfrecords
"""

import sys
sys.path.insert(0, '..')  # Set up sys path.

import datetime
import json
import random
import tensorflow.compat.v1 as tf
import math

from absl import app
from absl import flags
from functools import reduce
from google.cloud import storage
from io import StringIO
from shared.utils import *

FLAGS = flags.FLAGS
flags.DEFINE_string('src_bucket', None,
                    'GCS bucekt where exported resources are stored.')
flags.DEFINE_string(
    'src_folder', None,
    'GCS folder in the src_bucket where exported resources are stored.')
flags.DEFINE_string('dst_bucket', None,
                    'GCS bucket to save the tensowflow record file.')
flags.DEFINE_string(
    'dst_folder', None,
    'GCS folder in the dst_bucket to save the tensowflow record file.')
flags.mark_flag_as_required('src_bucket')
flags.mark_flag_as_required('dst_bucket')


def load_resources_by_type(res_type):
  client = storage.Client()
  bucket = client.get_bucket(FLAGS.src_bucket)
  for blob in bucket.list_blobs(prefix=FLAGS.src_folder):
    if blob.name.endswith(res_type):
      lines = blob.download_as_string().splitlines()
      return [json.loads(r.decode("utf-8")) for r in lines]
  return []


def build_examples():

  def map_by_id(result, item):
    result[item['id']] = item
    return result

  patients = reduce(map_by_id, load_resources_by_type(PATIENT_TYPE), {})
  questionnaire_responses = load_resources_by_type(QUESTIONNAIRERESPONSE_TYPE)

  def map_by_qid(result, item):
    qid = extract_uuid(extract_evidence_id(item))
    if qid in result:
      result[qid].append(item)
    else:
      result[qid] = [item]
    return result

  conditions = reduce(map_by_qid, load_resources_by_type(CONDITION_TYPE), {})

  examples = []
  for questionnaire_response in questionnaire_responses:
    pid = extract_uuid(questionnaire_response['subject']['reference'])
    if pid not in patients:
      continue

    patient = patients[pid]

    diseases = []
    qid = questionnaire_response['id']
    if qid in conditions:
      diseases = map(lambda x: extract_condition_disease(x), conditions[qid])

    age = calculate_age(patient['birthDate'])
    gender = 1 if patient['gender'] == 'male' else 0
    country = COUNTRY_MAP[extract_country(questionnaire_response)]
    duration = calculate_duration(
        *extract_start_end_date(questionnaire_response))

    for disease in DISEASE_MAP:
      risk = 1 if disease in diseases else 0

      feature = {
          'age':
              tf.train.Feature(int64_list=tf.train.Int64List(value=[age])),
          'gender':
              tf.train.Feature(int64_list=tf.train.Int64List(value=[gender])),
          'country':
              tf.train.Feature(int64_list=tf.train.Int64List(value=[country])),
          'duration':
              tf.train.Feature(int64_list=tf.train.Int64List(value=[duration])),
          'disease':
              tf.train.Feature(
                  int64_list=tf.train.Int64List(value=[DISEASE_MAP[disease]])),
          'risk':
              tf.train.Feature(int64_list=tf.train.Int64List(value=[risk])),
      }
      examples.append(
          tf.train.Example(features=tf.train.Features(feature=feature)))

  return examples


def save_examples(examples):
  """Splits examples into training and evaludate groups and saves to GCS."""
  random.shuffle(examples)
  # First 80% as training data, rest for evaluation.
  idx = int(math.ceil(len(examples) * .8))

  training_folder_path = "%s/%s" % (FLAGS.dst_folder, 'training.tfrecord') \
    if FLAGS.dst_folder is not None else 'training.tfrecord'
  record_path = "gs://%s/%s" % (FLAGS.dst_bucket, training_folder_path)
  with tf.io.TFRecordWriter(record_path) as w:
    for example in examples[:idx]:
      w.write(example.SerializeToString())

  eval_folder_path = "%s/%s" % (FLAGS.dst_folder, 'eval.tfrecord') \
    if FLAGS.dst_folder is not None else 'eval.tfrecord'
  record_path = "gs://%s/%s" % (FLAGS.dst_bucket, eval_folder_path)
  with tf.io.TFRecordWriter(record_path) as w:
    for example in examples[idx + 1:]:
      w.write(example.SerializeToString())


def main(_):
  save_examples(build_examples())


if __name__ == '__main__':
  app.run(main)
