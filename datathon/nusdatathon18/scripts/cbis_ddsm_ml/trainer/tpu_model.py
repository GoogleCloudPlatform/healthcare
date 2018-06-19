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

This model is built on top of the "2018 NUS-MIT Datathon Tutorial: Machine
Learning on CBIS-DDSM" tutorial (you can find it at https://git.io/vhgOu).
The architecture of the core model itself remains the same (i.e. 6 layers CNN).
However, substantial changes around data loading and metrics colletion have
been made to feed data to TPUs for training. For detailed explanation on how
the model works, please go to https://www.tensorflow.org/tutorials/layers.

This model asssumes that the data needed for training and evaluation has
already been generated and stored on GCS in TFRecords (more can be found:
https://www.tensorflow.org/programmers_guide/datasets). You can find a script
at https://git.io/vhg3K which helps you transform existing images to the
desired data format.

Please check out the tutorials folder (https://git.io/vhgaw) for instructions
on training this model with TPU.
"""

import tensorflow as tf

# Cloud TPU Cluster Resolver flags.
tf.flags.DEFINE_string(
  "tpu", default=None,
  help="The Cloud TPU to use for training. This should be either the name "
  "used when creating the Cloud TPU, or a grpc://ip.address.of.tpu:8470 "
  "url.")
tf.flags.DEFINE_string(
  "tpu_zone", default="us-central1-b",
  help="[Optional] GCE zone where the Cloud TPU is located in. At this "
  "moment, only us-central provides TPU access.")
tf.flags.DEFINE_string(
  "gcp_project", default=None,
  help="[Optional] Project name for the Cloud TPU-enabled project.")

# Input data specific flags.
tf.flags.DEFINE_string("training_data", default=None,
  help="Path to training data. This should be a GCS path, "
       "e.g. gs://cbis-ddsm-colab/cache/ddsm_train.tfrecords")
tf.flags.DEFINE_string("eval_data", default=None,
  help="Path to evaluation data. This should be a GCS path, "
       "e.g. gs://cbis-ddsm-colab/cache/ddsm_eval.tfrecords")
tf.flags.DEFINE_integer("image_width", default=0,
  help="Wdith of input images. All images are expected to share the same "
       "size.")
tf.flags.DEFINE_integer("image_height", default=0,
  help="Height of input images. All images are expected to share the same "
       "size.")
tf.flags.DEFINE_integer("image_channel", default=1,
  help="[Optional] Number of channels in input images.")

# Model specific flags.
tf.flags.DEFINE_string("model_dir", default=None,
  help="Estimator model_dir.")
tf.flags.DEFINE_integer("batch_size", default=96,
  help="Mini-batch size for the training. Note that this is the global batch "
       "size and not the per-shard batch.")
tf.flags.DEFINE_integer("training_steps", default=1000,
  help="Total number of training steps.")
tf.flags.DEFINE_integer("eval_steps", default=100,
  help="Total number of evaluation steps.")
tf.flags.DEFINE_float("learning_rate", default=0.05, help="Learning rate.")
tf.flags.DEFINE_integer("iterations", default=50,
  help="Number of iterations per TPU training loop.")
tf.flags.DEFINE_integer("num_shards", default=8,
  help="Number of shards (TPU chips).")
tf.flags.DEFINE_integer("category_count", default=0,
  help="Number of categories.")

FLAGS = tf.flags.FLAGS

def metric_fn(labels, logits):
  """Record metrics for evaluation."""
  predictions = tf.argmax(logits, 1)
  return {
    "accuracy": tf.metrics.precision(labels=labels, predictions=predictions)
  }

def cnn_model_fn(features, labels, mode, params):
  """CNN core model.

  Please read the tensorflow doc for how to customize this function:
  https://www.tensorflow.org/get_started/custom_estimators"""
  del params # Not needed.

  # Input Layer.
  # Reshape to 4-D tensor: [batch_size, height, width, channels]
  input_layer = tf.reshape(features,
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

  # Loss Calculation.
  loss = tf.losses.sparse_softmax_cross_entropy(labels=labels, logits=logits)

  if mode == tf.estimator.ModeKeys.TRAIN:
    learning_rate = tf.train.exponential_decay(
      FLAGS.learning_rate, tf.train.get_global_step(), 100000, 0.96)
    optimizer = tf.contrib.tpu.CrossShardOptimizer(
      tf.train.GradientDescentOptimizer(learning_rate=0.001))
    train_op = optimizer.minimize(
        loss=loss,
        global_step=tf.train.get_global_step())
    return tf.contrib.tpu.TPUEstimatorSpec(
      mode=mode, loss=loss, train_op=train_op)

  return tf.contrib.tpu.TPUEstimatorSpec(
    mode=mode,loss=loss, eval_metrics=(metric_fn, [labels, logits]))

def get_input_fn(filename):
  """Returns an `input_fn` for training and evaluation."""

  def input_fn(params):
    # Retrieves the batch size for the current shard. The number of shards is
    # computed according to the input pipeline deployment. See
    # https://www.tensorflow.org/api_docs/python/tf/contrib/tpu/RunConfig
    # for details.
    batch_size = params["batch_size"]

    def parse(serialized_example):
      """Parses a single tf.Example into image and label tensors."""
      features = tf.parse_single_example(
          serialized_example,
          features={
              "label": tf.FixedLenFeature([], tf.int64),
              "image": tf.FixedLenFeature([], tf.string),
          })
      image = tf.decode_raw(features["image"], tf.float32)
      image = tf.reshape(image, [FLAGS.image_width, FLAGS.image_height])
      label = tf.cast(features["label"], tf.int32)
      return image, label

    dataset = tf.data.TFRecordDataset(filename, buffer_size=500000)
    dataset = dataset.map(parse).cache().repeat()
    dataset = dataset.apply(
        tf.contrib.data.batch_and_drop_remainder(batch_size))
    images, labels = dataset.make_one_shot_iterator().get_next()
    return images, labels
  return input_fn

def main(_):
  """Set up training and evaluation steps."""
  tpu_cluster_resolver = tf.contrib.cluster_resolver.TPUClusterResolver(
    FLAGS.tpu,
    zone=FLAGS.tpu_zone,
    project=FLAGS.gcp_project
  )
  run_config = tf.contrib.tpu.RunConfig(
      cluster=tpu_cluster_resolver,
      model_dir=FLAGS.model_dir,
      session_config=tf.ConfigProto(
          allow_soft_placement=True, log_device_placement=True),
      tpu_config=tf.contrib.tpu.TPUConfig(FLAGS.iterations, FLAGS.num_shards),
  )
  classifier = tf.contrib.tpu.TPUEstimator(
      model_fn=cnn_model_fn,
      use_tpu=True,
      train_batch_size=FLAGS.batch_size,
      eval_batch_size=FLAGS.batch_size,
      config=run_config)

  # Set up logging for predictions.
  # Log the values in the "Softmax" tensor with label "probabilities".
  tensors_to_log = {"probabilities": "softmax_tensor"}
  logging_hook = tf.train.LoggingTensorHook(
      tensors=tensors_to_log, every_n_iter=50)

  # Training.
  classifier.train(
    input_fn=get_input_fn(FLAGS.training_data), steps=FLAGS.training_steps)

  # Evaluation.
  classifier.evaluate(
    input_fn=get_input_fn(FLAGS.eval_data), steps=FLAGS.eval_steps)

if __name__ == '__main__':
  # Set logging level to INFO.
  tf.logging.set_verbosity(tf.logging.INFO)
  tf.app.run()
