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
r"""A simple logistics regression model for immunization prediction.

The following features are used in this model:

1. age of the patient
2. BMI
3. smoker

We are predicting the possibility of the patient getting a disease.

Note that this model is part of an end-to-end demo which shows how
to leverage the Google Cloud Healthcare APIs (FHIR APIs specifically)
to finish data analysis and machine learning tasks. This problem
itself is not a natural machine learning task.
"""

import functools
import tensorflow.compat.v1 as tf

# Input data specific flags.
tf.flags.DEFINE_string(
    "training_data",
    default=None,
    help="Path to training data. This should be a GCS path.")
tf.flags.DEFINE_string(
    "eval_data",
    default=None,
    help="Path to evaluation data. This should be a GCS path.")

# Model specific flags. See more details here:
# https://www.tensorflow.org/api_docs/python/tf/estimator/LinearClassifier
tf.flags.DEFINE_string("model_dir", default=None, help="Estimator model_dir.")
tf.flags.DEFINE_string(
    "export_model_dir", default=None, help="Folder to export trained model.")
tf.flags.DEFINE_integer(
    "batch_size", default=96, help="Mini-batch size for the training.")
tf.flags.DEFINE_integer(
    "training_steps", default=1000, help="Total number of training steps.")
tf.flags.DEFINE_integer(
    "eval_steps", default=100, help="Total number of evaluation steps.")
tf.flags.DEFINE_integer(
    "n_classes", default=2, help="Number of categories to classify to.")

# More advanced flags that controls the behavior of FTRL optimizer.
# See more details here:
# https://www.tensorflow.org/api_docs/python/tf/train/FtrlOptimizer
tf.flags.DEFINE_float("learning_rate", default=0.01, help="Learning rate")
tf.flags.DEFINE_float(
    "l1_regularization_strength",
    default=0.005,
    help="L1 regularization strength for FTRL optimizer.")
tf.flags.DEFINE_float(
    "l2_regularization_strength",
    default=0.001,
    help="L2 regularization strength for FTRL optimizer.")

FLAGS = tf.flags.FLAGS

# Feature and label keys.
FEATURE_KEYS = ["age", "weight", "is_smoker"]
LABEL_KEYS = ["has_cancer"]

DS_BUFFER_SIZE = 50000


def _build_input_fn(filename):
  """Builds the input function for training/evaluation.

  Args:
    filename (string): The path of the file that contains features and
    labels. This can be a Google Cloud Storage path (e.g. gs://...).

  Returns:
    Function: Input function to be used by the classifier.
  """

  def input_fn():
    """Input function to be used by the classifier."""

    def parse(serialized_example):
      """Parses a single tensorflow example."""

      def parse_feature(features, key):
        features[key] = tf.FixedLenFeature([], tf.int64)
        return features

      data = tf.parse_single_example(
          serialized_example,
          features=functools.reduce(parse_feature, FEATURE_KEYS + LABEL_KEYS,
                                    {}))

      features = [
          tf.convert_to_tensor(tf.cast(data[key], tf.int32))
          for key in FEATURE_KEYS
      ]
      labels = [
          tf.convert_to_tensor(tf.cast(data[key], tf.int32))
          for key in LABEL_KEYS
      ]
      return features, labels

    dataset = tf.data.TFRecordDataset(filename, buffer_size=DS_BUFFER_SIZE)
    dataset = dataset.map(parse).cache().repeat()
    dataset = dataset.batch(FLAGS.batch_size)
    features, labels = dataset.make_one_shot_iterator().get_next()

    # Slice features into a dictionary which is expected by the classifier.
    features = tf.transpose(features)

    def map_feature(d, idx):
      """Maps individual features into a dictionary."""
      d[FEATURE_KEYS[idx]] = tf.transpose(
          tf.nn.embedding_lookup(features, [idx]))
      return d

    return functools.reduce(map_feature, list(range(len(FEATURE_KEYS))),
                            {}), labels

  return input_fn


def build_serving_input_receiver_fn():
  """Builds a serving_input_receiver_fn which takes JSON as input."""

  def serving_input_receiver_fn():

    def add_input(inputs, feature):
      inputs[feature] = tf.placeholder(shape=[None], dtype=tf.int32)
      return inputs

    inputs = functools.reduce(add_input, FEATURE_KEYS, {})
    return tf.estimator.export.ServingInputReceiver(inputs, inputs)

  return serving_input_receiver_fn


def main(_):
  # All features have been converted to integer representation beforehand.
  feature_columns = [
      tf.feature_column.numeric_column(key=key, dtype=tf.int32)
      for key in FEATURE_KEYS
  ]

  classifier = tf.estimator.LinearClassifier(
      feature_columns=feature_columns,
      model_dir=FLAGS.model_dir,
      n_classes=FLAGS.n_classes,
      optimizer=tf.train.FtrlOptimizer(
          learning_rate=FLAGS.learning_rate,
          l1_regularization_strength=FLAGS.l1_regularization_strength,
          l2_regularization_strength=FLAGS.l2_regularization_strength),
      config=tf.estimator.RunConfig(keep_checkpoint_max=1))

  # Training.
  classifier.train(
      input_fn=_build_input_fn(FLAGS.training_data), steps=FLAGS.training_steps)

  # Evaluation.
  classifier.evaluate(
      input_fn=_build_input_fn(FLAGS.eval_data), steps=FLAGS.eval_steps)

  # Export SavedModel.
  if FLAGS.export_model_dir is not None:
    classifier.export_saved_model(FLAGS.export_model_dir,
                                  build_serving_input_receiver_fn())


if __name__ == "__main__":
  # Set logging level to INFO.
  tf.logging.set_verbosity(tf.logging.INFO)
  tf.app.run()
