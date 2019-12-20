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
"""Preprocesses TCIA CBIS-DDSM dataset to enable Transfer Learning.

This script is a preprocessing step for the script that trains the model,
namely ml_codelab/scripts/trainer/model.py. This scripts builds an Apache
Beam pipeline on Cloud Dataflow to scalably pre-process the images.

This script will download the labels from TCIA website
(https://wiki.cancerimagingarchive.net/display/Public/CBIS-DDSM) and will
split the input dataset (specified by --image_dir) into a training,
validation and test sets. The percentage of each can be specified by using
the --validation_percentage and --testing_percentage flags.

The output of this script is TFRecords containing the "bottleneck" for each
image. A "bottleneck" is the feature vector output from the checkpoint model (in
this case Inception V3). We are calculating the bottlenecks so they can be fed
to the classification part of the model in model.py. The bottlenecks represent
the output of the feature extraction part of the model.
"""

import warnings
import argparse
import csv
import logging
import os
import random
import StringIO
import sys
import threading
import apache_beam as beam
from apache_beam.options.pipeline_options import PipelineOptions
import httplib2
import scripts.constants as constants
import scripts.tcia_utils as tcia_utils
import numpy as np

# TODO: Remove when Tensorflow library is updated.
warnings.filterwarnings('ignore')
import tensorflow as tf
from tensorflow.python.lib.io import file_io

# Labels for BI-RADS breast density scores.
_BREAST_DENSITY_2_LABEL = '2'
_BREAST_DENSITY_3_LABEL = '3'


class PreprocessGraph(object):
  """ Creates a TF graph to preprocess an image and to calculate bottlenecks.
  Example usage:
  # Create the Tensorflow graph.
  preprocess_graph = PreprocessGraph(sess)

  # Calculate bottlenecks for an input image.
  bottleneck = preprocess_graph.calculate_bottleneck(input_image)
  return bottleneck
  """

  def __init__(self):
    graph = tf.Graph()
    self._tf_session = tf.Session(graph=graph)
    with graph.as_default():
      self._input_jpeg_tensor, self._bottleneck_tensor = self._build_graph()
      self._tf_session.run(tf.global_variables_initializer())

  def _build_graph(self):
    # type: None -> (tf.Tensor, tf.Tensor)
    """Builds the processing graph.

    Returns:
      (input_jpeg_str, bottleneck_tensor) tuple.

      input_jpeg_str is a Tensor for input JPEG image.
      bottleneck_tensor is a Tensor for output bottleneck Tensor.
    """

    input_jpeg_str = tf.placeholder(tf.string)
    # Make ml_utils a local import. This means that ml_utils does not have
    # to be installed on machine that starts the workers, it only needs to be
    # installed on the workers themselves. This makes dependency management a
    # bit easier.
    # https://cloud.google.com/dataflow/faq
    import scripts.ml_utils as ml_utils
    bottleneck_tensor = ml_utils.get_bottleneck_tensor(input_jpeg_str)
    return input_jpeg_str, bottleneck_tensor

  def calculate_bottleneck(self, image_bytes):
    # type: str -> np.ndarray
    """Returns the bottleneck for an image."""

    bottleneck = self._tf_session.run(
        self._bottleneck_tensor,
        feed_dict={self._input_jpeg_tensor: image_bytes})
    return np.squeeze(bottleneck)


def _to_tfrecord(dataset, image_path, label, bottleneck):
  # type: (str, str, str, List[float]) -> tensorflow.train.Example
  """Converts input into a TFRecord.

  Args:
    dataset: String indicator of dataset - training, validation or test.
    image_path: Path of input image in GCS
    label: Label associated with the image.
    bottleneck: The bottleneck vector.

  Returns:
    A tf.train.Example object.
  """

  def _bytes_feature(value):
    return tf.train.Feature(bytes_list=tf.train.BytesList(value=[value]))

  def _float_feature(value):
    return tf.train.Feature(float_list=tf.train.FloatList(value=value))

  example = tf.train.Example(
      features=tf.train.Features(
          feature={
              'dataset': _bytes_feature(dataset),
              'image_path': _bytes_feature(image_path),
              'label': _bytes_feature(label),
              'bottleneck': _float_feature(bottleneck)
          }))
  return example


class PreprocessImage(beam.DoFn):
  """Workflow step to preprocess input images.

  This workflow step does the following:
  1) Reads input image from GCS
  2) Resize and encodes the image as required by Inception V3 model.
  3) Calculate Inception V3 bottleneck and stores it as a TFRecord.
  """

  # Synchronization for Beam variables that can be called from multiple threads.
  # https://groups.google.com/a/google.com/forum/#!msg/dataflow-beam-portability/MQ4UqpDhwyg/Cal4yAniAgAJ
  _preprocess_graph_lock = threading.Lock()
  _preprocess_graph = None

  def start_bundle(self):
    # type: None -> None
    """Starts an Apache Beam bundle.

    We cache the Tensorflow session per bundle to avoid cold starts for the
    processing of each element.
    """
    with self._preprocess_graph_lock:
      if self._preprocess_graph is None:
        self._preprocess_graph = PreprocessGraph()

  def process(self, element):
    # type: beam.PCollection -> Iterable[tensorflow.TFRecord]
    """Calculates the bottleneck for an image.

    Args:
      element: A beam.PCollection holding the (dataset, image_path, label).

    Yields:
      TFRecord holding (image_path, label, bottleneck).

    Raises:
      RuntimeError: If _preprocess_graph is not initialized.
    """
    (dataset, image_path, label) = element
    image_data = file_io.FileIO(image_path, 'rb').read()
    if self._preprocess_graph is None:
      raise RuntimeError('self._preprocess_graph not initialized')
    bottleneck = self._preprocess_graph.calculate_bottleneck(image_data)
    yield _to_tfrecord(dataset, image_path, label, bottleneck)


def _get_study_uid_to_image_path_map(input_path):
  # type: str -> Dict[str, str]
  """Helper to run a map of study_uid to image path.

  Args:
    input_path: Input path of images.

  Returns:
    study_uid_to_file_paths: Dictionary mapping study_uid to image path.
  """
  path_list = file_io.get_matching_files(os.path.join(input_path, '*/*/*'))
  study_uid_to_file_paths = {}
  for path in path_list:
    split_path = path.split('/')
    study_uid_to_file_paths[split_path[3]] = path
  return study_uid_to_file_paths


def _partition_fn(element, unused_num_partitions):
  dataset = element.features.feature['dataset'].bytes_list.value[0]
  if dataset == constants.TRAINING_DATASET:
    return 0
  elif dataset == constants.VALIDATION_DATASET:
    return 1
  return 2


def configure_pipeline(p, opt):
  # Type: (apache_beam.Pipeline, apache_beam.PipelineOptions) -> None
  """Specify PCollection and transformations in pipeline."""
  # Create a map of study_uid to label.
  study_uid_to_label = tcia_utils.GetStudyUIDToLabelMap()

  # Create a map of study_uid -> path of images in GCS
  study_uid_to_image_path = _get_study_uid_to_image_path_map(opt.input_path)

  # Create a map of study_uid -> (GCS path, label)
  # Split dataset into training, validation and test.
  paths_and_labels = []
  dataset_size = len(study_uid_to_label)
  training_size = dataset_size * (
      100 - opt.testing_percentage - opt.validation_percentage) / 100
  validation_size = dataset_size * opt.validation_percentage / 100
  testing_size = dataset_size * opt.testing_percentage / 100
  logging.info('Number of images in training dataset: %s', training_size)
  logging.info('Number of images in validation dataset: %s', validation_size)
  logging.info('Number of images in testing dataset: %s', testing_size)

  count = 0
  for k, v in study_uid_to_label.iteritems():
    if k not in study_uid_to_image_path:
      logging.warning('Could not find image with study_uid %s in GCS', k)
      continue
    if count < training_size:
      dataset = constants.TRAINING_DATASET
    elif count >= training_size and count < (training_size + validation_size):
      dataset = constants.VALIDATION_DATASET
    else:
      dataset = constants.TESTING_DATASET
    count += 1
    paths_and_labels.append((dataset, study_uid_to_image_path[k], v))

  # Shuffle the input
  random.shuffle(paths_and_labels)
  parts = (
      p
      | 'Download Labels' >> beam.Create(paths_and_labels)
      | 'Preprocess Image' >> beam.ParDo(PreprocessImage())
      | 'Split into Training-Validation-Testing' >> beam.Partition(
          _partition_fn, 3))

  # Branch into workflows that serialize training/validation/testing TFRecords.
  for idx, path_suffix in enumerate([
      constants.TRAINING_DATASET, constants.VALIDATION_DATASET,
      constants.TESTING_DATASET
  ]):
    _ = (
        parts[idx]
        | 'Serialize TFRecord ' + path_suffix >>
        beam.Map(lambda x: x.SerializeToString())
        | 'Save TFRecord to GCS ' + path_suffix >> beam.io.WriteToTFRecord(
            os.path.join(opt.output_path, path_suffix),
            file_name_suffix='.tfrecord'))


def run(in_args=None):
  """Runs the pre-processing pipeline."""

  pipeline_options = PipelineOptions.from_dictionary(vars(in_args))
  with beam.Pipeline(options=pipeline_options) as p:
    configure_pipeline(p, in_args)


def default_args(argv):
  """Provides default values for Workflow flags."""
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--project',
      required=True,
      type=str,
      help='The cloud project ID to be used for running this pipeline')
  parser.add_argument(
      '--job_name',
      type=str,
      default='breast-density-preprocessing',
      help='A unique job identifier.')
  parser.add_argument(
      '--num_workers', default=5, type=int, help='The number of workers.')
  parser.add_argument(
      '--input_path',
      required=True,
      help=
      'Path to input directory of images. The images are expected to be in a GCS bucket with format gs://<bucket_name>/<study_uid>/<series_uid>/<instance_uid>.jpg'
  )
  parser.add_argument(
      '--output_path',
      required=True,
      help='Output directory to write results TFRecords to.')
  parser.add_argument(
      '--temp_location',
      required=True,
      help=
      'Temp location for workflow. See https://cloud.google.com/dataflow/pipelines/specifying-exec-params for details.'
  )
  parser.add_argument(
      '--staging_location',
      required=True,
      help=
      'Staging location for workflow. See https://cloud.google.com/dataflow/pipelines/specifying-exec-params for details.'
  )
  parser.add_argument(
      '--testing_percentage',
      type=int,
      default=10,
      help='What percentage of images to use as a test set.')
  parser.add_argument(
      '--validation_percentage',
      type=int,
      default=10,
      help='What percentage of images to use as a validation set.')

  parser.add_argument('--cloud', default=True, action='store_true')
  parser.add_argument(
      '--runner',
      help='See Dataflow runners, may be blocking'
      ' or not, on cloud or not, etc.')

  parsed_args, _ = parser.parse_known_args(argv)

  if parsed_args.cloud:
    # Flags which need to be set for cloud runs.
    default_values = {
        'runner': 'DataflowRunner',
        'save_main_session': True,
        'worker_machine_type': 'n1-standard-4',
        'setup_file': './setup.py',
        'autoscaling_algorithm': 'NONE',
    }
  else:
    # Flags which need to be set for local runs.
    default_values = {
        'runner': 'DirectRunner',
    }

  for kk, vv in default_values.iteritems():
    if kk not in parsed_args or not vars(parsed_args)[kk]:
      vars(parsed_args)[kk] = vv

  return parsed_args


def main(argv):
  arg_dict = default_args(argv)
  run(arg_dict)


if __name__ == '__main__':
  main(sys.argv[1:])
  print('Successfully preprocessed images!')
