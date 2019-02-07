"""Convert DICOM png images a (sharded) TFRecord.

An Apache Beam pipeline for processing .png images that were exported by a
Google Cloud Healthcare (CHC) dicomStore into a TFRecord suitable for ingestion
into Google Cloud ML Engine.

Notes:
* This script is only compatible with python2.7 since
Apache Beam currently only supports python2.7
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import re
import apache_beam as beam
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.options.pipeline_options import SetupOptions
import concurrent.futures
import tensorflow as tf
from google.cloud import storage

# The path format for .png images exported by Cloud Healthcare DICOM Stores.
# See `https://cloud.google.com/healthcare/docs/reference/rest/v1alpha/
# projects.locations.datasets.dicomStores/export#gcsdestination`
PATH_REGEX = r'.*/([^/]+)/([^/]+)/([^/]+)\.png$'


class GetPaths(beam.DoFn):
  """Collect all the paths in a GCS bucket with a given prefix.

  Notes:
    Since this operation is a "fan-out" type operation, it should generally
    be followed by a beam.ReShuffle transform to avoid spurious fusion.

  Args:
    bucket (str): the GCS bucket to search
    max_results (Union[int, None], default: None): the maximum number of paths
      to yield per prefix
    validation_regex (Union[str, None], default: None):
      a regular expression that paths, excluding the gs:/{bucket}/ prefix, must
        match in order to be added to the output PCollection
  """

  def __init__(self, bucket, max_results=None, validation_regex=None):
    super(GetPaths, self).__init__()
    self.bucket = bucket
    self.max_results = max_results
    # Not serializable, to be initialized on each worker:
    self._client = None
    self.uid_regex = re.compile(validation_regex) if validation_regex else None

  def process(self, prefix):
    """Overrides beam.DoFn.process.

    Args:
      prefix (str): the prefix that generated paths share

    Yields:
      str: all GCS blob URIs that start with the string
      gs://{self.bucket}/{prefix} and match validation_regex if it was specified
    """
    # Since google.cloud.storage.Clients are not pickle-able
    # this pattern allows them to be initialized on each worker exactly once.
    if self._client is None:
      self._client = storage.Client()
    bucket = self._client.bucket(self.bucket)
    for path in bucket.list_blobs(prefix=prefix, max_results=self.max_results):
      if self.uid_regex is None or self.uid_regex.match(path.name):
        yield 'gs://{}/{}'.format(self.bucket, path.name)


class ReadCHCPngFiles(beam.DoFn):
  """Reads the entire contents of png files exported by the CHC API to GCS.

  Extracts the study, series and instance unique identifiers form the file path
  and reads the binary contents into memory.

  Args:
    max_threads (Optional[int]): the number of threads to use for parallel file
      reads. The default value is 4.
  """

  def __init__(self, max_threads=4):
    self.max_threads = max_threads
    self.futures = set()
    self.executor = None
    self.uid_regex = re.compile(PATH_REGEX)

  def process(self, element):
    """Overrides beam.DoFn.process.

    Args:
      element (str): the path to a DICOM instance exported to a png image by
        Google CHC. See https://cloud.google.com/healthcare/docs/reference/rest/
          v1alpha/projects.locations.datasets.dicomStores/export#gcsdestination
          for the format of these paths.

    Yields:
      Tuple[Tuple[str, str, str], bytes]:
        the (study, series, instance) UIDs for this DICOM instance, and
        the binary contents of the .png file.
    """
    if self.executor is None:
      self.executor = concurrent.futures.ThreadPoolExecutor(self.max_threads)
    self.futures.add(self.executor.submit(self.download, element))
    # Maintain an input queue approximately 4 times as large as the number
    # of workers to keep the workers busy
    if len(self.futures) > 4 * self.max_threads:
      done, self.futures = concurrent.futures.wait(
          self.futures, return_when=concurrent.futures.FIRST_COMPLETED)
      for future in done:
        path, file_bytes = future.result()
        yield self.uid_regex.match(path).groups(), file_bytes

  def finish_bundle(self):
    """Overrides beam.DoFn.finish_bundle."""
    for future in concurrent.futures.as_completed(self.futures):
      path, file_bytes = future.result()
      ans = (self.uid_regex.search(path).groups(), file_bytes)
      yield beam.utils.windowed_value.WindowedValue(
          ans, beam.utils.timestamp.MIN_TIMESTAMP,
          [beam.transforms.window.GlobalWindow()])
    self.futures.clear()

  def download(self, path):
    with tf.gfile.GFile(path, 'rb') as fp:
      file_bytes = fp.read()
    return path, file_bytes


class ResizePng(beam.DoFn):
  """Resize an image that is represented as a png encoded binary bytestring.

  Args:
    height (int): the height to resize the images to.
    width (int): the width to reisze the images to.
    color_channels (int, default=None): the number of colour channels to use to
      represent the image. The default behaviour is to use the image's native
      representation.
  """

  def __init__(self, height, width, color_channels=None):
    super(ResizePng, self).__init__()

    self.image_shape = (height, width)
    if color_channels is None:
      # Match the color channels of the input image using tf.image.decode_png.
      self.image_channels = 0
    else:
      self.image_channels = color_channels

    self.initialized = False
    # Not serializable, to be initialized on each worker
    self._input_png_bytes_tensor = None
    self._output_png_bytes_tensor = None
    self._session = None

  def initialize(self):
    """Initialize the tensorflow graph and session for this worker."""
    # Input GCS file path
    self._input_png_bytes_tensor = tf.placeholder(tf.string, [])
    # Decode the png_bytes into a HWC array
    u8image = tf.image.decode_png(
        self._input_png_bytes_tensor, channels=self.image_channels)
    resized_u8image = tf.cast(
        tf.image.resize_images(u8image, self.image_shape), tf.uint8)
    self._output_png_bytes_tensor = tf.image.encode_png(resized_u8image)
    self._session = tf.Session()
    self.initialized = True

  def process(self, element):
    """Overrides beam.DoFn.process.

    Args:
      element (Tuple[Tuple[str, str, str], path]): the UIDs and path of the
        input image

    Yields:
      np.array: a HWC array of the resized image.
    """
    (study, series, instance), png_bytes = element
    if not self.initialized:
      # Initialize non-serializable data once on each worker
      self.initialize()
    png_bytes = self._session.run(self._output_png_bytes_tensor,
                                  {self._input_png_bytes_tensor: png_bytes})
    yield (study, series, instance), png_bytes


def to_tf_example(element):
  """Convert a (keys, data) pair into a tensorflow example.

  Args:
    element (Tuple[Tuple[str, str, str], bytes]): ((study, series, instance),
      png_bytes) The UIDs and binary png data for a dicom instance

  Returns:
    tensorflow.core.example.example_pb2.Example:
      A corresponding tensorflow example.
  """
  ((study, series, instance), png_bytes) = element

  def _bytes_feature(value):
    return tf.train.Feature(bytes_list=tf.train.BytesList(value=[value]))

  example = tf.train.Example(
      features=tf.train.Features(
          feature={
              'study': _bytes_feature(study),
              'series': _bytes_feature(series),
              'instance': _bytes_feature(instance),
              'png_bytes': _bytes_feature(png_bytes)
          }))
  return example


def run(bucket,
        input_gcs_path_prefix,
        output_name,
        pipeline_options,
        output_shape=None,
        output_channels=None):
  """Build and execute the Apache Beam pipeline.

  Args:
    bucket (str): The bucket that holds the input files.
    input_gcs_path_prefix (str): The common prefix for all the .png files
      processed. This does not include gs:// or bucket name or a leading slash.
        Only .png files with suffix {study_uid}/{series_uid}/{instance_uid}.png,
        i.e. the path format output by Cloud Healthcare DICOM export (see
      https://cloud.google.com/healthcare/docs/reference/rest/v1alpha/
        projects.locations.datasets.dicomStores/export#gcsdestination) will be
        processed.
    output_name (str): The path to the output TFRecord file, ending in the
      prefix used for the file name. If this pipeline is being run on Cloud
      Dataflow,
      this must begin with gs:// to specify GCS.
    pipeline_options (PipelineOptions): Apache Beam specific command line
      arguments.
      see https://beam.apache.org/documentation/runners/dataflow/
    output_shape (Optional[Tuple[int, int]]): The (height, width) to resize
      images to. If this is not provided, no resizing will be performed.
    output_channels (Optional[int]): The number of color channels to use for the
      output image. If this is not provided, then the image's default number of
      channels will be used.
  """
  with beam.Pipeline(options=pipeline_options) as p:
    png_bytes_pcoll = \
        (p
         | beam.Create([input_gcs_path_prefix])
         | 'GetPaths' >> beam.ParDo(GetPaths(bucket,
                                             validation_regex=PATH_REGEX))
         # Materialize and re-bundle paths with ReShuffle to enable parallelism.
         | beam.Reshuffle()
         | beam.ParDo(ReadCHCPngFiles(max_threads=4)))

    if output_shape is not None or output_channels is not None:
      png_bytes_pcoll |= beam.ParDo(
          ResizePng(*output_shape, color_channels=output_channels))

    _ = (
        png_bytes_pcoll
        | 'ConvertToTFExample' >> beam.Map(to_tf_example)
        | beam.io.WriteToTFRecord(
            output_name,
            file_name_suffix='.tfrecord',
            coder=beam.coders.ProtoCoder(tf.train.Example)))


if __name__ == '__main__':
  parser = argparse.ArgumentParser(description=__doc__)
  parser.add_argument(
      '--gcs_input_prefix',
      required=True,
      help="""A path on Google Cloud Storage beginning with gs://. All the files
      with this prefix that end in {study_id}/{instance_id}/{series_id}.png
      will be converted into tensorflow examples with the following
      byte-fields: "study", "series", "instance", "png_bytes". This is the
      the path format for .png images exported by Cloud Healthcare DICOM Stores.
      See `https://cloud.google.com/healthcare/docs/reference/rest/v1alpha/
      projects.locations.datasets.dicomStores/export#gcsdestination` for details.
      These files can be generated with prepare_dicom.py.""")

  parser.add_argument(
      '--output_image_shape',
      required=False,
      help="""A string specification of the form H,W or H,W,C (where H, W and
      C are all integers) that describes the dimensions to resize the images
      to. No resizing is performed if argument is not passed.""")

  parser.add_argument(
      '--output_prefix',
      required=True,
      help="""The path prefix to use for naming the output of this pipeline,
      which may be sharded across multiple files. This should be a gs:// path
      if you're using DataflowRunner.""")

  args, pipeline_args = parser.parse_known_args()
  beam_options = PipelineOptions(pipeline_args)
  # serialize and provide global imports, functions, etc. to workers.
  beam_options.view_as(SetupOptions).save_main_session = True

  if (beam_options.runner == 'DataflowRunner' and
      not args.output_prefix.startswith('gs://')):
    parser.error('--output_prefix must start with gs:// when using'
                 'DataflowRunner.')
  # Parse input path:
  # Match either just a bucket `gs://bucket` in which case path_prefix ==
  # '' or Match a bucket and a path prefix `gs://bucket/path_prefix`
  path_match = re.match(r'^gs://([a-z0-9\-_.]+)(|/.*)$', args.gcs_input_prefix)
  if path_match:
    input_bucket, path_prefix = path_match.groups()
    if path_prefix.startswith('/'):
      path_prefix = path_prefix[1:]
  else:
    parser.error('Invalid input for --gcs_input_prefix ' +
                 args.gcs_input_prefix)

  # Parse new image dimensions
  shape = None
  channels = None
  if args.output_image_shape is not None:
    hwc = [int(x) for x in args.output_image_shape.split(',')]
    if len(hwc) < 2 or len(hwc) > 3:
      parser.error('Invalid input for --output_image_shape ' +
                   args.output_image_shape)
    elif len(hwc) == 2:
      shape = hwc
    else:
      shape = hwc[:2]
      channels = hwc[2]

  run(input_bucket,
      path_prefix,
      args.output_prefix,
      beam_options,
      output_shape=shape,
      output_channels=channels)
