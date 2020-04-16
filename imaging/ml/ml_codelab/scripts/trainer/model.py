# Copyright 2018 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Trains breast density classifier on TCIA CBIS-DDSM using Transfer Learning.

This script assumes that the input images have already been preprocessed (by
ml_codelab/scripts/preprocess/preprocess.py). So this should be executed after
that script.

This script roughly does the following:
1) Reads bottleneck values (in TFRecord format).

   This will read all the bottlenecks found in --bottleneck_dir. The bottlenecks
   consist of a list of (image_path, label, bottleneck) tuples.

2) Split dataset into training-validation-test.

   This will split every record found in the input TFRecords into either the
   training, validation or test dataset. The percentage of each can be specified
   by using the --validation_percentage and --testing_percentage flags.

2) Add dense layer and softmax layer for classification.

   Add a dense and a softmax layer that classify mammography images as
   either "2" (scattered density) or "3" (heterogeneously dense).

4) Runs training.

   The script will run Tensorflow training on the model. There are various
   training parameters than can be tweaked, including the number of training
   steps (--how_many_training_steps), the learning rate (--learning_rate) and
   the training batch size (--train_batch_size).

   Every so often, the training accuracy, the cross-entropy (loss rate) and the
   validation accuracy will be logged to console.

5) Evalute test dataset.

   The script will finally output the accuracy of the trained model on the
   test dataset. If --print_misclassified_test_images is specified, it will
   additionally print the misclassified images in the test dataset.

6) Creates and exports serving model.

   The inference model consisting of the Inception V3 feature extraction layers
   and the dense/softmax classification is created. This model is stored in
   --export_model_path. This model uses the pre-trained Inception V3 weights,
   and the newly trained weights for the dense layer for breast density
   classification.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import collections
import datetime
import logging
import os
import sys
import scripts.constants as constants
import scripts.ml_utils as ml_utils
import tensorflow.compat.v1 as tf

FLAGS = None

# Have logging log to stdout
logging.basicConfig(stream=sys.stdout, level=logging.INFO)

# Size of bottleneck vector for Inception V3.
INCEPTION_V3_BOTTLENECK_SIZE = 2048

# File to store model checkpoint containing just the weights and biases of
# dense layer.
_CHECKPOINT_FILE = '/tmp/model.ckpt'


def _get_image_label_info(bottleneck_dir):
  # type: str -> (int, List[str])
  """Calculates the number of images and unique labels in dataset.

  Args:
    bottleneck_dir: Directory containing the bottleneck TFRecords.

  Returns:
    image_count: Total number of images in dataset.
    label_list: List of labels found in dataset.
  This function will parse the TFRecords found in bottleneck dir and will
  only return the labels.
  """
  labels = collections.OrderedDict()
  dataset_to_image_count = collections.defaultdict(int)
  bottleneck_files = tf.io.gfile.glob(os.path.join(bottleneck_dir, '*'))
  for bottleneck_file in bottleneck_files:
    for it in tf.io.tf_record_iterator(bottleneck_file):
      example = tf.train.Example()
      example.ParseFromString(it)
      label = example.features.feature['label'].bytes_list.value[0]
      labels[label] = True
      dataset = example.features.feature['dataset'].bytes_list.value[0]
      dataset_to_image_count[dataset] += 1

  label_list = []
  for key in labels:
    label_list.append(key)
  return dataset_to_image_count, label_list


def _get_training_validation_testing_dataset(bottleneck_dir, label_table,
                                             testing_dataset_size):
  """Gets the dataset for training, validation and testing.

  Args:
    bottleneck_dir: Directory containing the bottleneck TFRecords.
    label_table: A tensorflow Table of all labels found in dataset.
    testing_dataset_size: The number of images in the testing dataset.


  Returns:
    (training_dataset, validation_dataset, testing_dataset) tuple.

    training_dataset is a Dataset containing training images.
    validation_dataset is a Dataset containing validation images.
    testing_dataset is a Dataset containing testing images.
  """

  def _full_tfrecord_parser(serialized_example):
    """Parses a tf.Example into (image, label index, bottleneck) Tensors."""
    features = {
        'image_path':
            tf.FixedLenFeature((), tf.string),
        'label':
            tf.FixedLenFeature((), tf.string),
        'bottleneck':
            tf.FixedLenFeature([INCEPTION_V3_BOTTLENECK_SIZE], tf.float32),
    }
    example = tf.parse_single_example(serialized_example, features=features)
    label_index = label_table.lookup(example['label'])
    return example['image_path'], label_index, example['bottleneck']

  training_bottleneck_files = tf.io.gfile.glob(
      os.path.join(bottleneck_dir, constants.TRAINING_DATASET + '*'))
  training_dataset = tf.data.TFRecordDataset(training_bottleneck_files).map(
      _full_tfrecord_parser).repeat().batch(FLAGS.train_batch_size)

  validation_bottleneck_files = tf.io.gfile.glob(
      os.path.join(bottleneck_dir, constants.VALIDATION_DATASET + '*'))
  validation_dataset = tf.data.TFRecordDataset(validation_bottleneck_files).map(
      _full_tfrecord_parser).repeat().batch(FLAGS.validation_batch_size)

  testing_bottleneck_files = tf.io.gfile.glob(
      os.path.join(bottleneck_dir, constants.TESTING_DATASET + '*'))
  testing_dataset = tf.data.TFRecordDataset(testing_bottleneck_files).map(
      _full_tfrecord_parser).batch(testing_dataset_size)
  return training_dataset, validation_dataset, testing_dataset


def _build_signature(inputs, outputs):
  # type: (Dict[str, tf.Tensor], Dict[str, tf.Tensor]) -> (tensorflow.SignatureDef)
  """Build the Tensorflow serving signature.

  Args:
    inputs: A dictionary of tensor name to tensor.
    outputs: A dictionary of tensor name to tensor.

  Returns:
    The signature, a SignatureDef proto.
  """
  signature_inputs = {
      key: tf.saved_model.utils.build_tensor_info(tensor)
      for key, tensor in inputs.items()
  }
  signature_outputs = {
      key: tf.saved_model.utils.build_tensor_info(tensor)
      for key, tensor in outputs.items()
  }

  signature_def = tf.saved_model.signature_def_utils.build_signature_def(
      signature_inputs, signature_outputs,
      tf.saved_model.signature_constants.CLASSIFY_METHOD_NAME)

  return signature_def


def _generate_var_list(sess):
  # type: tensorflow.Session -> List[tensorflow.Variable]
  """Lists the checkpoint Tensors found in the current Tensorflow session.

  This function will map the tensor names found in _CHECKPOINT_FILE to
  Tensors found in this session. _CHECKPOINT_FILE contains the weight and
  biases for the dense layer saved earlier.

  Args:
    sess: A tensorflow session.

  Returns:
    variables: List of Tensors to be restored.
  """
  variables = []
  reader = tf.train.NewCheckpointReader(_CHECKPOINT_FILE)
  variable_to_shape_map = reader.get_variable_to_shape_map()
  for key in variable_to_shape_map:
    tensor = sess.graph.get_tensor_by_name(key + ':0')
    variables.append(tensor)
  return variables


def _export_model(label_list, export_model_path):
  # type: (List[str], str) -> None
  """Exports model for serving.

  Args:
    label_list: List of labels used for training.
    export_model_path: Path to store exported model used for inference.
  """
  class_count = len(label_list)

  # Create the inference graph.
  with tf.Graph().as_default() as graph:
    # Map from label index to label string.
    index_list = list(i for i in range(len(label_list)))
    initializer = tf.lookup.KeyValueTensorInitializer(
        index_list, label_list, key_dtype=tf.int64, value_dtype=tf.string)
    index_to_label_table = tf.lookup.StaticHashTable(
        initializer, default_value='UNKNOWN')
    # Inference graph, consisting of the Inception V3 feature extraction layers
    # and a dense and softmax layer used for classification.
    input_jpeg_str, label, score = _create_inference_graph(
        index_to_label_table, class_count)

  with tf.Session(graph=graph) as sess:
    # Initialize global variables.
    sess.run(tf.global_variables_initializer())

    # Restore the checkpoint. The checkpoint has the weights and biases for
    # the dense layer.
    tf.train.Saver(_generate_var_list(sess)).restore(sess, _CHECKPOINT_FILE)

    # The input and output of the inference model.
    inputs = {
        tf.saved_model.signature_constants.CLASSIFY_INPUTS: input_jpeg_str,
    }
    outputs = {
        tf.saved_model.signature_constants.CLASSIFY_OUTPUT_CLASSES: label,
        tf.saved_model.signature_constants.CLASSIFY_OUTPUT_SCORES: score,
    }

    # Default signature.
    # The model is avaiable via 'serving_default' signature name.
    signature_def = _build_signature(inputs=inputs, outputs=outputs)
    signature_def_map = {
        tf.saved_model.signature_constants.DEFAULT_SERVING_SIGNATURE_DEF_KEY:
            signature_def
    }

    # Save out the SavedModel.
    builder = tf.saved_model.builder.SavedModelBuilder(export_model_path)
    builder.add_meta_graph_and_variables(
        sess, [tf.saved_model.tag_constants.SERVING],
        assets_collection=tf.get_collection(tf.GraphKeys.ASSET_FILEPATHS),
        signature_def_map=signature_def_map,
        main_op=tf.group(tf.tables_initializer(), name='main_op'))
    builder.save()


def _create_dense_and_softmax_layers(
    bottleneck_tensor,
    class_count,
    output_softmax_tensor_name=constants.OUTPUT_SOFTMAX_TENSOR_NAME):
  # type: (tf.Tensor, int, str) -> (tf.Tensor, tf.Tensor, tf.Tensor)
  """Creates a dense and softmax layer.

  Args:
    bottleneck_tensor: Bottleneck values fed into dense layer.
    class_count: Number of classes used for classification.
    output_softmax_tensor_name: Name of softmax output tensor. This is used by
      the Explainable AI feature to show how output changes with input.

  Returns:
    (logits, final_tensor, prediction_index) tuple.

    logits is the output of the dense layer.
    final_tensor is the output of the softmax layer.
    prediction_index is the index of final_tensor that has highest score.
  """
  logits = tf.layers.dense(bottleneck_tensor, units=class_count)
  final_tensor = tf.nn.softmax(logits, name=output_softmax_tensor_name)
  prediction_index = tf.argmax(final_tensor, 1)
  return logits, final_tensor, prediction_index


def _create_inference_graph(index_to_label_table, class_count):
  # type: (tf.Tensor, int) -> (tf.Tensor, tf.Tensor, tf.Tensor)
  """Creates graph used for inference.

  Args:
    index_to_label_table: Table that maps index to label.
    class_count: Number of classes used for classification.

  Returns:
    (img_bytes, prediction, score) tuple.

    img_bytes is the input JPEG image.
    prediction is the predicted label.
    score is the score for the predicted label.
  """

  img_bytes, bottleneck_tensor = ml_utils.create_fixed_weight_input_graph()
  _, normalized_tensor, prediction_index = _create_dense_and_softmax_layers(
      bottleneck_tensor, class_count)
  # Get the prediction (label) for a given tensor index.
  prediction = index_to_label_table.lookup(prediction_index)
  # Get the score for the predicted class.
  score = tf.squeeze(tf.gather(normalized_tensor, prediction_index, axis=1))
  return img_bytes, prediction, score


def _create_training_graph(bottleneck_tensor, label_index_tensor, class_count,
                           learning_rate):
  # type: (tf.Tensor, tf.Tensor, int, int) -> (tf.Tensor, tensorflow.Operation, tf.Tensor, tf.Tensor)
  """Creates a Tensorflow graph used for training.

  Args:
    bottleneck_tensor: Tensor for bottleneck values of image.
    label_index_tensor: Tensor to look up labels by label indices.
    class_count: Total number of classes.
    learning_rate: The learning rate for training.

  Returns:
    (cross_entropy_mean, train_step, prediction, evaluation_step) tuple.

    cross_entropy_mean is Tensor that holds cross entropy mean.
    train_step is a Tensorflow operation that runs training step.
    prediction_index is a Tensor that holds index with the largest value across
    axes.
    evaluation_step is a Tensor that computes the mean of elements across
     dimensions of a tensor, used when evaluating the model.
  """
  logits, _, prediction_index = _create_dense_and_softmax_layers(
      bottleneck_tensor, class_count)
  cross_entropy_mean = tf.losses.sparse_softmax_cross_entropy(
      labels=label_index_tensor, logits=logits)
  optimizer = tf.train.GradientDescentOptimizer(learning_rate)
  train_step = optimizer.minimize(cross_entropy_mean)
  is_correct_prediction = tf.equal(prediction_index, label_index_tensor)
  evaluation_step = tf.reduce_mean(tf.cast(is_correct_prediction, tf.float32))
  return cross_entropy_mean, train_step, prediction_index, evaluation_step


def main(_):
  # Tensorflow Hub logging is very verbose, only log errors and above.
  tf.logging.set_verbosity(tf.logging.ERROR)

  # Check that the bottleneck directory exists.
  assert tf.io.gfile.exists(
      FLAGS.bottleneck_dir), ('bottleneck dir %s missing' %
                              FLAGS.bottleneck_dir)

  # Find the total number of images and labels
  dataset_counter, label_list = _get_image_label_info(FLAGS.bottleneck_dir)
  class_count = len(label_list)
  logging.info('=== IMAGE STATISTICS ===')
  logging.info('Number of classes: %s', class_count)
  for k, v in dataset_counter.items():
    logging.info('Number of images for %s dataset: %s', k, v)

  # Maps from labels to index and vice-versa.
  index_list = list(i for i in range(len(label_list)))
  initializer = tf.lookup.KeyValueTensorInitializer(
      label_list, index_list, value_dtype=tf.int64)
  label_to_index_table = tf.lookup.StaticVocabularyTable(
      initializer, num_oov_buckets=1)
  # Split dataset into training, validation and testing
  training_dataset, validation_dataset, testing_dataset = (
      _get_training_validation_testing_dataset(
          FLAGS.bottleneck_dir, label_to_index_table,
          dataset_counter[constants.TESTING_DATASET.encode()]))

  # Create iterators for the training, validation and testing dataset.
  bottleneck_input = tf.placeholder(tf.string)
  iterator = tf.data.Iterator.from_string_handle(bottleneck_input,
                                                 training_dataset.output_types,
                                                 training_dataset.output_shapes)
  image_path, label_index, bottleneck = iterator.get_next()
  training_iter = training_dataset.make_initializable_iterator()
  validation_iter = validation_dataset.make_initializable_iterator()
  testing_iter = testing_dataset.make_initializable_iterator()

  # Create the graph containing the dense and softmax layer.
  cross_entropy_mean, train_step, prediction, evaluation_step = _create_training_graph(
      bottleneck, label_index, class_count, FLAGS.learning_rate)

  # Train the graph and run evaluation.
  train_saver = tf.train.Saver()
  with tf.Session() as sess:
    # Initialize global variables.
    sess.run(tf.global_variables_initializer())
    # Initialize table that maps label to label_index.
    sess.run(tf.tables_initializer())

    # Initialize feedable iterators, these feed directly into feed_dict of ops.
    training_handle = sess.run(training_iter.string_handle())
    validation_handle = sess.run(validation_iter.string_handle())
    testing_handle = sess.run(testing_iter.string_handle())

    # Initializable iterators required an explicit initialization before use.
    # https://www.tensorflow.org/guide/datasets#consuming_values_from_an_iterator
    sess.run(training_iter.initializer)
    sess.run(validation_iter.initializer)

    # Training loop. Runs through --training_steps iterations of training. Every
    # so often, it will output the accuracy on both the training and
    # validation dataset.
    for i in range(FLAGS.training_steps):
      # Run training iteration.
      sess.run(train_step, feed_dict={bottleneck_input: training_handle})

      # Run validation iteration every --eval_step_interval iterations. Output
      # the up to date training and validation accuracy.
      is_last_step = (i + 1 == FLAGS.training_steps)
      if (i % FLAGS.eval_step_interval) == 0 or is_last_step:
        training_accuracy, cross_entropy_value = sess.run(
            [evaluation_step, cross_entropy_mean],
            feed_dict={bottleneck_input: training_handle})
        logging.info('%s: Step %d: Train accuracy = %.1f%%',
                     datetime.datetime.now(), i, training_accuracy * 100)
        logging.info('%s: Step %d: Cross entropy = %f', datetime.datetime.now(),
                     i, cross_entropy_value)
        validation_accuracy = sess.run(
            evaluation_step, feed_dict={bottleneck_input: validation_handle})
        logging.info('%s: Step %d: Validation accuracy = %.1f%%',
                     datetime.datetime.now(), i, validation_accuracy * 100)

    # Save the model variables.
    train_saver.save(sess, _CHECKPOINT_FILE)

    # Calculate accuracy on test dataset.
    sess.run(testing_iter.initializer)
    testing_accuracy, predictions, image_paths, labels_indexes = sess.run(
        [evaluation_step, prediction, image_path, label_index],
        feed_dict={bottleneck_input: testing_handle})
    logging.info('Final test accuracy = %.1f%%', testing_accuracy * 100)

    # Print misclassified images.
    if FLAGS.print_misclassified_test_images:
      logging.info('=== MISCLASSIFIED TEST IMAGES ===')
      for i, image_path in enumerate(image_paths):
        if predictions[i] != labels_indexes[i]:
          logging.info('%s:\nPrediction: %s\nGround Truth: %s', image_path,
                       label_list[predictions[i]],
                       label_list[labels_indexes[i]])

  # Export the model for inference.
  _export_model(label_list, FLAGS.export_model_path)


if __name__ == '__main__':
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--bottleneck_dir',
      type=str,
      default='',
      help='Path to folder containing bottlenecks in TFRecord format.')
  parser.add_argument(
      '--training_steps',
      type=int,
      default=1000,
      help='How many training steps to run before ending.')
  parser.add_argument(
      '--learning_rate',
      type=float,
      default=0.01,
      help='How large a learning rate to use when training.'
  )
  parser.add_argument(
      '--eval_step_interval',
      type=int,
      default=10,
      help='How often to evaluate the training results.'
  )
  parser.add_argument(
      '--train_batch_size',
      type=int,
      default=100,
      help='How many images to train on at a time.'
  )
  parser.add_argument(
      '--validation_batch_size',
      type=int,
      default=100,
      help="""\
      How many images to use in an evaluation batch. This validation set is
      used much more often than the test set, and is an early indicator of how
      accurate the model is during training.
      A value of -1 causes the entire validation set to be used, which leads to
      more stable results across training iterations, but may be slower on large
      training sets.\
      """
  )
  parser.add_argument(
      '--print_misclassified_test_images',
      default=False,
      help="""\
      Whether to print out a list of all misclassified test images.\
      """,
      action='store_true')
  parser.add_argument(
      '--export_model_path',
      default='/tmp/inference/1',
      help="""\
      Directory to store the saved inference model. Note that a version number
      is required in the end of the path (e.g. '1' in the default). This number
      is needed for severing Tensorflow via tensorflow_model_server tool.
      """)
  FLAGS, unparsed = parser.parse_known_args()
  tf.app.run(main=main, argv=[sys.argv[0]] + unparsed)
