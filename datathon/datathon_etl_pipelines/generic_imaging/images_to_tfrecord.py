"""Generate a TFRecord Dataset from JPG or PNG images and BigQuery.

This pipeline joins a collection of JPG or PNG images stored in Google Cloud
Storage, with a BigQuery query statement and produces a set of labelled images
in TFRecord format.

The provided BigQuery statement must generate a column named `path` which
contains a list of GCS uris (beginning with gs://) that is used to join images
with labels.

The output TFRecord has features:
- (jpg|png)_bytes (bytes): the encoded image
- path (bytes): the path to the original image
and a feature for every additional column in the input query.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse

import apache_beam as beam
from apache_beam.options.pipeline_options import GoogleCloudOptions
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.options.pipeline_options import SetupOptions
from datathon_etl_pipelines.dofns.get_paths import GetPaths
from datathon_etl_pipelines.dofns.resize_image import ResizeImage
from datathon_etl_pipelines.utils import ArgumentValidator
from datathon_etl_pipelines.utils import bytes_feature
from datathon_etl_pipelines.utils import get_setup_file
from datathon_etl_pipelines.utils import scalar_to_feature
import tensorflow as tf
from google.cloud import bigquery


def download(path):
  with tf.io.gfile.GFile(path, 'rb') as fp:
    return path, fp.read()


def key_on_path(row):
  labels = row.copy()
  path = labels.pop('path')
  return path, labels


class ConvertToTFExample(beam.DoFn):
  """Convert joined labels and images into TFExamples.

  Args:
    image_format (Union['jpg', 'png']): The format of the images.
  """

  def __init__(self, image_format):
    if image_format == 'jpeg':
      image_format = 'jpg'
    if image_format not in ('jpg', 'png'):
      raise ValueError('Unrecognized image format ' + image_format)
    self.image_format = image_format

  def process(self, element):
    """Construct a tf.train.Example from an image joined with labels.

    Args:
      element (Tuple[unicode, Dict[Union['images', 'labels'], List[Dict]]]): The
        set of ids for the image along with the joined image and labels.

    Yields:
      tf.train.Example: All pairs of images with labels.
    """
    path, join = element

    for image in join['images']:
      for labels in join['labels']:
        features = {
            'path': bytes_feature(path.encode()),
            self.image_format + '_bytes': bytes_feature(image)
        }
        for name, value in labels.viewitems():
          features[name] = scalar_to_feature(value)

        example = tf.train.Example(features=tf.train.Features(feature=features))
        yield example


def build_and_run_pipeline(pipeline_options, gcs_input_prefix, image_format,
                           output_image_shape, path_match_regex, bigquery_query,
                           output_tfrecord_prefix):
  """Construct and run the Apache Beam Pipeline.

  Args:
    pipeline_options (PipelineOptions): Passed to Apache Beam.
    gcs_input_prefix (List[str]): One or more paths on Google Cloud Storage
      beginning with gs://. Specifies the image files to operate on.
    image_format (Union['jpg', 'png']): The format of the input images.
    output_image_shape (Iterable[int]): The dimensions to resize the image to.
      Either HW or HWC. If this is None, then the images will not be resized.
    path_match_regex (str): A regex that the GCS paths of images must match if
      they are to be processed.
    bigquery_query (str): A StandardSQL BigQuery query that produces the labels
      for the images. This query must output one column named path, which is
      used to join the labels with the images (using the image's full GCS path,
      beginning with gs://). All the other columns are used as labels in the
        output TFRecord dataset.
    output_tfrecord_prefix: The full GCS path to the output TFRecord, including
      the prefix of the object names, with will have shard identifies and
      .tfrecord appended to them.
  """
  with beam.Pipeline(options=pipeline_options) as p:
    images = (
        p
        | beam.Create(gcs_input_prefix)
        | 'GetPaths' >> beam.ParDo(GetPaths(validation_regex=path_match_regex))
        # Materialize and re-bundle paths with ReShuffle to enable parallelism.
        | beam.Reshuffle()
        | beam.Map(download))

    if output_image_shape is not None:
      images |= beam.ParDo(ResizeImage(image_format, *output_image_shape))

    labels = (
        p
        | 'ReadTable' >> beam.io.Read(
            beam.io.BigQuerySource(query=bigquery_query, use_standard_sql=True))
        | beam.Map(key_on_path))

    joined = {'images': images, 'labels': labels} | beam.CoGroupByKey()

    _ = (
        joined
        | beam.ParDo(ConvertToTFExample(image_format))
        | beam.io.WriteToTFRecord(
            output_tfrecord_prefix,
            file_name_suffix='.tfrecord',
            coder=beam.coders.ProtoCoder(tf.train.Example)))


class GcsInputPrefixValidator(ArgumentValidator):
  """Validate the --gcs_input_prefix argument."""

  def validate_value(self, gcs_input_prefix, **kwargs):
    for prefix in gcs_input_prefix:
      if not prefix.startswith('gs://'):
        self.raise_exception('{} does not begin with gs://'.format(prefix))


class QueryValidator(ArgumentValidator):
  """Validate the --query argument."""

  def validate_value(self, query, **kwargs):
    client = bigquery.Client(project=kwargs.get('project', None))
    query_job = client.query('SELECT * FROM ({}) LIMIT 10'.format(query))
    for row in query_job.result():
      if 'path' not in row.keys():
        self.raise_exception(
            'the provided query does not generate a column named `path`.')
      if not row['path'].startswith('gs://'):
        self.raise_exception(
            'the provided query generated a value in the path column that did '
            'not begin with gs://: {}'.format(row['path']))


def main():
  parser = argparse.ArgumentParser(description=__doc__)
  validators = list()
  validators.append(
      GcsInputPrefixValidator(
          parser.add_argument(
              '--gcs_input_prefix',
              nargs='+',
              required=True,
              help='One or more paths on Google Cloud Storage beginning with '
              'gs://. Specifies the image files to operate on.')))
  parser.add_argument(
      '--image_format',
      choices=['png', 'jpg'],
      required=True,
      help='The format of the input images.')
  parser.add_argument(
      '--output_image_shape',
      required=False,
      nargs='+',
      type=int,
      help='The dimensions to resize the image to. Either HW or HWC. If this '
      'is None, then the images will not be resized.')
  parser.add_argument(
      '--path_match_regex',
      required=False,
      help='A regex that the GCS paths of images must match if they are to be '
      'processed.')
  validators.append(
      QueryValidator(
          parser.add_argument(
              '--query',
              required=True,
              help='A StandardSQL BigQuery query that produces the labels for '
              'the  images. This query must output one column named `path`, '
              'which is used to join the labels with the images (using the '
              'image\'s full GCS path, beginning with gs://). All the other '
              'columns are used as labels in the output TFRecord dataset.')))
  parser.add_argument(
      '--output_tfrecord_prefix',
      required=False,
      help='The full GCS path to the output TFRecord, including the prefix of'
      'the object names, with will have shard identifies and .tfrecord '
      'appended to them.')
  args, pipeline_args = parser.parse_known_args()
  beam_options = PipelineOptions(pipeline_args)
  for validator in validators:
    validator.validate(
        args, project=beam_options.view_as(GoogleCloudOptions).project)
  beam_options.view_as(SetupOptions).save_main_session = True
  beam_options.view_as(SetupOptions).setup_file = get_setup_file()

  if args.output_image_shape is not None:
    if len(args.output_image_shape) not in (2, 3):
      parser.error('2 (HW) or 3 (HWC) integers are required for '
                   'output_image_shape')

  build_and_run_pipeline(
      pipeline_options=beam_options,
      gcs_input_prefix=args.gcs_input_prefix,
      image_format=args.image_format,
      output_image_shape=args.output_image_shape,
      path_match_regex=args.path_match_regex,
      bigquery_query=args.query,
      output_tfrecord_prefix=args.output_tfrecord_prefix)


if __name__ == '__main__':
  main()
