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
r"""Loads Synthea Patient bundles in GCS and extracts relevant features and labels from the resources for training.

The genereated tensorflow record dataset is stored in a specified location on
GCS.

Example usage:
python assemble_training_data.py --src_bucket=my-bucket \
                                 --src_folder=export    \
                                 --dst_bucket=my-bucket \
                                 --dst_folder=tfrecords
"""

import gzip
import json
import math
import os
import random
import tempfile

from absl import app
from absl import flags
from shared import features

import tensorflow.compat.v1 as tf
from google.cloud import storage

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


def _load_examples_from_gcs():
  """Downloads the examples from a GCS bucket."""
  client = storage.Client()
  bucket = storage.Bucket(client, FLAGS.src_bucket)
  for blob in bucket.list_blobs(prefix=FLAGS.src_folder):
    if not blob.name.endswith('.gz'):
      continue
    print('Downloading patient record file', blob.name)
    with tempfile.NamedTemporaryFile() as compressed_f:
      blob.download_to_filename(compressed_f.name)
      print('Building TF records')
      with gzip.open(compressed_f.name, 'r') as f:
        for line in f:
          yield features.build_example(json.loads(line.decode('utf-8')))


def _load_examples_from_file():
  """Loads the examples from a gzipped-ndjson file."""
  for file in os.listdir(FLAGS.src_folder):
    if not file.endswith('.gz'):
      continue
    with gzip.open(os.path.join(FLAGS.src_folder, file), 'r') as f:
      for line in f:
        yield features.build_example(json.loads(line))


def _build_examples():
  """Loads the examples and converts the features into TF records."""
  examples = []
  patients = iter([])
  if not FLAGS.src_bucket:
    patients = _load_examples_from_file()
  else:
    patients = _load_examples_from_gcs()
  for patient in patients:
    if patient is None:
      continue

    feature = {
        'age':
            tf.train.Feature(
                int64_list=tf.train.Int64List(value=[patient['age']])),
        'weight':
            tf.train.Feature(
                int64_list=tf.train.Int64List(value=[patient['weight']])),
        'is_smoker':
            tf.train.Feature(
                int64_list=tf.train.Int64List(value=[patient['is_smoker']])),
        'has_cancer':
            tf.train.Feature(
                int64_list=tf.train.Int64List(value=[patient['has_cancer']])),
    }
    examples.append(
        tf.train.Example(features=tf.train.Features(feature=feature)))

  return examples


def _save_examples(examples):
  """Splits examples into training and evaludate groups and saves to GCS."""
  random.shuffle(examples)
  # First 80% as training data, rest for evaluation.
  idx = int(math.ceil(len(examples) * .8))

  training_folder_path = FLAGS.dst_folder + '/training.tfrecord' if FLAGS.dst_folder is not None else 'training.tfrecord'
  record_path = 'gs://{}/{}'.format(FLAGS.dst_bucket, training_folder_path)
  with tf.io.TFRecordWriter(record_path) as w:
    for example in examples[:idx]:
      w.write(example.SerializeToString())

  eval_folder_path = FLAGS.dst_folder + '/eval.tfrecord' if FLAGS.dst_folder is not None else 'eval.tfrecord'
  record_path = 'gs://{}/{}'.format(FLAGS.dst_bucket, eval_folder_path)
  with tf.io.TFRecordWriter(record_path) as w:
    for example in examples[idx + 1:]:
      w.write(example.SerializeToString())


def main(_):
  _save_examples(_build_examples())


if __name__ == '__main__':
  app.run(main)
