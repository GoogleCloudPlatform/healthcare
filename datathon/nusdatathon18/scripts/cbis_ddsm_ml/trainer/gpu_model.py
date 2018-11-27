#!/usr/bin/python
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

"""
A simple CNN model for classifying CBIS-DDSM images based on breast density
(https://breast-cancer.ca/densitbi-rads/) categories.

This model is basically the same as what we have in the "2018 NUS-MIT Datathon
Tutorial: Machine Learning on CBIS-DDSM" tutorial (you can find it at
https://git.io/vhgOu). For detailed explanation on how the model works, please
go to https://www.tensorflow.org/tutorials/layers.
"""

import numpy as np
import pandas as pd
import tensorflow as tf

from google.cloud import storage
from io import BytesIO
from PIL import Image

# Input data specific flags.
tf.flags.DEFINE_string("data_bucket", default=None,
  help="The bucket where input data is stored.")
tf.flags.DEFINE_string("training_data_dir", default=None,
  help="Path to training data within the data bucket.")
tf.flags.DEFINE_string("eval_data_dir", default=None,
  help="Path to evaluation data within the data bucket.")
tf.flags.DEFINE_integer("image_width", default=0,
  help="Wdith of input images.")
tf.flags.DEFINE_integer("image_height", default=0,
  help="Height of input images.")
tf.flags.DEFINE_integer("image_channel", default=1,
  help="[Optional] Number of channels of input images.")

# Model specific flags.
tf.flags.DEFINE_string("model_dir", default=None,
  help="Estimator model_dir")
tf.flags.DEFINE_integer("batch_size", default=96,
  help="Mini-batch size for the training.")
tf.flags.DEFINE_integer("training_steps", default=1000,
  help="Total number of training steps.")
tf.flags.DEFINE_integer("eval_steps", default=100,
  help="Total number of evaluation steps.")
tf.flags.DEFINE_integer("category_count", default=0,
  help="Number of categories.")

FLAGS = tf.flags.FLAGS

class ImageLoader:
  """Loading training and evaluation images from GCS."""
  def __init__(self):
    self._client = storage.Client()
    self._bucket = self._client.get_bucket(FLAGS.data_bucket)

  def load_train_images(self):
    return self._load_images(FLAGS.training_data_dir)

  def load_test_images(self):
    return self._load_images(FLAGS.eval_data_dir)

  def _load_images(self, folder):
    """Loads images from a GCS bucket.

    Returns a tuple of an array of numpy matrices and an array of labels.
    """
    images = []
    labels = []
    for label in [1, 2, 3, 4]:
      blobs = self._bucket.list_blobs(prefix=("%s/%s_" % (folder, label)))

      for blob in blobs:
        byte_stream = BytesIO()
        blob.download_to_file(byte_stream)
        byte_stream.seek(0)

        img = Image.open(byte_stream)
        images.append(np.array(img, dtype=np.float32))
        labels.append(label-1) # Minus 1 to fit in [0, 4).

    return np.array(images), np.array(labels, dtype=np.int32)

def cnn_model_fn(features, labels, mode):
  """CNN core model.

  Please read the tensorflow doc for how to customize this function:
  https://www.tensorflow.org/get_started/custom_estimators"""

  # Input Layer.
  # Reshape to 4-D tensor: [batch_size, height, width, channels]
  input_layer = tf.reshape(features["image"],
    [-1, FLAGS.image_height, FLAGS.image_width, FLAGS.image_channel])

  # Convolutional Layer #1.
  conv1 = tf.layers.conv2d(
      inputs=input_layer,
      filters=32,
      kernel_size=[5, 5],
      padding="same",
      activation=tf.nn.relu)

  # Pooling Layer #1.
  pool1 = tf.layers.max_pooling2d(inputs=conv1, pool_size=[2, 2], strides=2)

  # Convolutional Layer #2.
  conv2 = tf.layers.conv2d(
      inputs=pool1,
      filters=64,
      kernel_size=[5, 5],
      padding="same",
      activation=tf.nn.relu)

  # Pooling Layer #2.
  pool2 = tf.layers.max_pooling2d(inputs=conv2, pool_size=[2, 2], strides=2)

  # Flatten tensor into a batch of vectors
  filtered_width = FLAGS.image_width / 4
  filtered_height = FLAGS.image_height / 4
  pool2_flat = tf.reshape(pool2, [-1, filtered_width * filtered_height * 64])

  # Dense Layer.
  dense = tf.layers.dense(inputs=pool2_flat, units=1024, activation=tf.nn.relu)

  # Dropout operation.
  dropout = tf.layers.dropout(inputs=dense, rate=0.4,
                              training=(mode == tf.estimator.ModeKeys.TRAIN))

  # Logits Layer.
  logits = tf.layers.dense(inputs=dropout, units=FLAGS.category_count)

  predictions = {
      # Generate predictions (for PREDICT and EVAL mode)
      "classes": tf.argmax(input=logits, axis=1),
      # Add `softmax_tensor` to the graph. It is used for PREDICT and by the
      # `logging_hook`.
      "probabilities": tf.nn.softmax(logits, name="softmax_tensor")
  }
  if mode == tf.estimator.ModeKeys.PREDICT:
    return tf.estimator.EstimatorSpec(mode=mode, predictions=predictions)

  # Loss Calculation.
  loss = tf.losses.sparse_softmax_cross_entropy(labels=labels, logits=logits)

  if mode == tf.estimator.ModeKeys.TRAIN:
    optimizer = tf.train.GradientDescentOptimizer(learning_rate=0.001)
    train_op = optimizer.minimize(
        loss=loss,
        global_step=tf.train.get_global_step())
    return tf.estimator.EstimatorSpec(mode=mode, loss=loss, train_op=train_op)

  # Add evaluation metrics (for EVAL mode).
  eval_metric_ops = {
      "accuracy": tf.metrics.accuracy(
          labels=labels, predictions=predictions["classes"])}
  return tf.estimator.EstimatorSpec(
      mode=mode, loss=loss, eval_metric_ops=eval_metric_ops)

def main(_):
  # Load training and test data.
  image_loader = ImageLoader()
  train_data, train_labels = image_loader.load_train_images()
  eval_data, eval_labels = image_loader.load_test_images()

  # Create the Estimator.
  classifier = tf.estimator.Estimator(
      model_fn=cnn_model_fn,
      model_dir=FLAGS.model_dir)

  # Set up logging for predictions.
  # Log the values in the "Softmax" tensor with label "probabilities".
  tensors_to_log = {"probabilities": "softmax_tensor"}
  logging_hook = tf.train.LoggingTensorHook(
      tensors=tensors_to_log, every_n_iter=50)

  # Train the model.
  train_input_fn = tf.compat.v1.estimator.inputs.numpy_input_fn(
      x={"image": train_data},
      y=train_labels,
      batch_size=FLAGS.batch_size,
      num_epochs=None,
      shuffle=True)
  classifier.train(
      input_fn=train_input_fn,
      steps=FLAGS.training_steps,
      hooks=[logging_hook])

  # Evaluate the model and print results.
  eval_input_fn = tf.compat.v1.estimator.inputs.numpy_input_fn(
      x={"image": eval_data}, y=eval_labels, num_epochs=1, shuffle=False)
  eval_results = classifier.evaluate(
    input_fn=eval_input_fn,
    steps=FLAGS.eval_steps)
  print(eval_results)

if __name__ == '__main__':
  # Set logging level to INFO.
  tf.logging.set_verbosity(tf.logging.INFO)
  tf.app.run()
