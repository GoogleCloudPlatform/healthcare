"""Build an Apache Beam pipeline for keras inference to BigQuery."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import tempfile

import apache_beam as beam
from apache_beam.io.gcp.bigquery import BigQueryDisposition
from apache_beam.io.gcp.bigquery import WriteToBigQuery
from apache_beam.io.tfrecordio import ReadFromTFRecord
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.options.pipeline_options import SetupOptions
from datathon_etl_pipelines.utils import get_setup_file
import numpy as np
import tensorflow as tf


def default_normalize(image):
  return (1.0 / 255.0) * tf.cast(image, tf.float32)


class ExampleWithImageBytesToInput(object):
  """Convert a TFExample into an image to input into Model.

  Args:
    image_format (Union['jpg', 'png']): the format of the images to decode.
    image_process_fn (Callable[[tf.Tensor], tf.Tensor]): A function to apply to
      the decoded image. Defaults to dividing by 255.0 to normalize the pixel
      intensities to be in [0, 1].
    feature_name (Optional[str]): the name of the feature that contains the JPG
      or PNG bytes to decode. Defaults to (jpg|png)_bytes, according to
      image_format.
  """

  def __init__(self,
               image_format,
               image_process_fn=default_normalize,
               feature_name=None):
    if image_format == 'jpeg':
      image_format = 'jpg'
    if image_format not in ('jpg', 'png'):
      raise ValueError('Unrecognized image format ' + image_format)
    if feature_name is None:
      self.feature_name = '{}_bytes'.format(image_format)
    else:
      self.feature_name = feature_name
    self.image_format = image_format
    self._image_process_fn = image_process_fn
    self._session = None
    self._input_bytes_tensor = None
    self._image = None
    self.initialized = False

  def initialize(self):
    """Initialize the tensorflow graph and session on this worker."""
    self._input_bytes_tensor = tf.placeholder(tf.string, [])
    if self.image_format == 'jpg':
      decode_fn = tf.image.decode_jpeg
    elif self.image_format == 'png':
      decode_fn = tf.image.decode_png
    else:
      raise ValueError('Unrecognized image format ' + self.image_format)
    self._image = self._image_process_fn(decode_fn(self._input_bytes_tensor))
    self._session = tf.Session()
    self.initialized = True

  def __call__(self, example):
    """Convert a TFExample into a array of pixel intensities.

    Args:
      example (tf.train.Example): A TFExample with a 'jpg_bytes' string feature.

    Returns:
      np.array: a HWC array of the image, with pixel intensities in [0, 1].
    """
    if not self.initialized:
      # Initialize non-serializable data once on each worker
      self.initialize()

    image = self._session.run(
        self._image, {
            self._input_bytes_tensor:
                example.features.feature[self.feature_name].bytes_list.value[0]
        })
    return image


class Predict(beam.DoFn):
  """Perform inference on a single example.

  Args:
    keras_model_uri (str): The GCS uri, beginning with gs://, or a local path.
      This specifies the saved Keras model to use for inference. This model is
      created with `tf.keras.models.Model.save`.
    example_to_row (Callable[[tf.train.Example], Dict[str, Any]]): A function
      that maps the input tf.train.Example to a dictionary of column names and
      values, which identify each instance.
    example_to_input(Callable[[tf.train.Example], np.array]): A function that
      maps the input tf.train.Example to a numpy array. This array is the input
      for the model. The batch dimension should not be included.
    output_to_row(Callable[[tf.train.Example], Dict[str, Any]]): A function that
      maps the output of the model to a dictionary of column names and values.
      This is combined with the ids to form the output of this DoFn.

  Returns:
    Dict[str, Any]: A dictionary identifiers and output labels. Maps column
    names to column values.
  """

  def __init__(self, keras_model_uri, example_to_row, example_to_input,
               output_to_row):
    super(Predict, self).__init__()
    self.keras_model_uri = keras_model_uri
    self._example_to_row = example_to_row
    self._example_to_input = example_to_input
    self._output_to_row = output_to_row

    self.initialized = False
    self._model = None

  def initialize(self):
    """Initialize the keras model on this worker."""
    # Download the model from GCS. If the model is a local file, this also
    # doesn't hurt.
    with tempfile.NamedTemporaryFile('wb') as local_model_file:
      with tf.io.gfile.GFile(self.keras_model_uri, 'rb') as gcs_model_file:
        local_model_file.write(gcs_model_file.read())
      local_model_file.flush()
      # Keras models don't need to be compiled for inference.
      self._model = tf.keras.models.load_model(
          local_model_file.name, compile=False)
    self.initialized = True

  def process(self, element):
    """Overrides beam.DoFn.process.

    Args:
      element (tf.train.Example): The instance to run inference on.

    Yields:
      Dict[str, Any]: A column-name -> value dictionary, with ids and prediction
        results.
    """
    if not self.initialized:
      self.initialize()
    output_row = self._example_to_row(element)

    input_array = np.expand_dims(self._example_to_input(element), axis=0)
    output_array = self._model.predict_on_batch(input_array)
    # remove batch dimension before passing to self._output_to_row
    output_row.update(self._output_to_row(output_array[0, ...]))
    yield output_row


def build_and_run_pipeline(pipeline_options, tfrecord_pattern, predict_dofn,
                           output_bq_table, bq_table_schema):
  """Build and run a Keras batch inference pipeline to BigQuery pipeline.

  Args:
    pipeline_options (beam.options.pipeline_options import PipelineOptions):
      Commandline arguments for this pipeline.
    tfrecord_pattern (str): A file glob pattern to read TFRecords from.
    predict_dofn (beam.DoFn): A DoFn that transforms TFExamples into
      dictionaries describing BigQuery rows.
    output_bq_table (str): A string of the form `project:dataset.table_name`.
      This table will be overwritten if it already exists.
    bq_table_schema (Union[str, TableSchema]): A BigQuery schema in the format
      used by `apache_beam.io.gcp.bigquery.WriteToBigQuery`.
  """
  with beam.Pipeline(options=pipeline_options) as p:
    _ = (
        p
        | ReadFromTFRecord(
            tfrecord_pattern, coder=beam.coders.ProtoCoder(tf.train.Example))
        | beam.ParDo(predict_dofn)
        | WriteToBigQuery(
            table=output_bq_table,
            schema=bq_table_schema,
            write_disposition=BigQueryDisposition.WRITE_TRUNCATE))


def get_commandline_args(description):
  """Generate command line arguments used by inference to BigQuery scripts.

  Args:
    description (str): The description of the script, which will appear at the
      top of the --help documentation.

  Returns:
    Tuple[Namespace, PipelineOptions]: 1) The commandline options with fields
      `input_tfrecord_pattern`, `keras_model`, and `bigquery_table` 2) The
      apache beam pipeline options.
  """
  parser = argparse.ArgumentParser(description=description)
  parser.add_argument(
      '--input_tfrecord_pattern',
      type=str,
      help='A file glob pattern that specifies the TFRecord files to read from.'
  )
  parser.add_argument(
      '--keras_model',
      type=str,
      help='The GCS uri, beginning with gs://, or a local path. This specifies '
      'the saved Keras model to use for inference. This model is created '
      'with `tf.keras.models.Model.save`.')
  parser.add_argument(
      '--bigquery_table',
      type=str,
      help='The table to store the labelled predictions in. This is a string '
      'of the form `project:dataset.table_name`. This table will be '
      'overwritten if it already exists.')
  args, pipeline_args = parser.parse_known_args()
  beam_options = PipelineOptions(pipeline_args)
  beam_options.view_as(SetupOptions).save_main_session = True
  beam_options.view_as(SetupOptions).setup_file = get_setup_file()
  return args, beam_options
