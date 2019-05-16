r"""Decompress files in GCS Storage using GCS Transcoding.

See https://cloud.google.com/storage/docs/transcoding for details on the
eligibility of input files.

Common usage patterns are:

Removing the .gz suffix on decompressed files:
--path_sub_pattern='(.*)\.gz'  --path_sub_repl='\1'

Placing the decompressed files in a new folder (called folder_uz):
--path_sub_pattern='(.*)/folder/(.*)\.gz' --path_sub_repl='\1/folder_uz/\2'

Overwrite the original file:
--path_sub_pattern='(.*)' --path_sub_repl='\1'
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import re

import apache_beam as beam
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.options.pipeline_options import SetupOptions
from datathon_etl_pipelines.dofns.get_paths import GetPaths
from datathon_etl_pipelines.dofns.get_paths import split_into_bucket_and_blob
from datathon_etl_pipelines.utils import get_setup_file
import tensorflow as tf
from google.cloud import storage


class DecompressAndWrite(beam.DoFn):
  r"""Decompresses files using GCS transcoding.

  See https://cloud.google.com/storage/docs/transcoding for details on the
  eligibility of input files.

  Args:
    pattern (str): the regular expression pattern to match the paths of the
      uncompressed files with.
    repl (str): the string to replace matches of the pattern with. This may
      include regular expression capture groups: \1, \2, etc. The resulting
        string is used for the path of the uncompressed file.
  """

  def __init__(self, pattern, repl):
    super(DecompressAndWrite, self).__init__()
    self.sub_regex = re.compile(pattern)
    self.repl = repl
    # Not serializable, to be initialized on each worker:
    self._client = None

  def process(self, path):
    """Overrides beam.DoFn.process."""
    old_path = path
    old_bucket_name, old_object_name = split_into_bucket_and_blob(path)
    new_path = self.sub_regex.sub(self.repl, old_path)

    if self._client is None:
      self._client = storage.Client()

    with tf.gfile.GFile(new_path, 'wb') as fp:
      self._client.bucket(old_bucket_name).blob(
          old_object_name).download_to_file(fp)


def main():
  """Build and execute the Apache Beam pipeline using the commandline arguments."""
  parser = argparse.ArgumentParser(description=__doc__)
  parser.add_argument(
      '--gcs_input_prefix',
      nargs='+',
      required=True,
      help="""One or more paths on Google Cloud Storage beginning with gs://.
      All the files with this prefix that match the path_sub_pattern regex will
      be decompressed.""")

  parser.add_argument(
      '--path_sub_pattern',
      required=True,
      help="""The regular expression pattern that file paths must match in order
      to be decompressed. This is regular expression is also used to generate
      the name of the new uncompressed file, by replacing the matching portion
      with path_sub_repl.""")

  parser.add_argument(
      '--path_sub_repl',
      required=True,
      help=r"""the string to replace the match in the original path with to
      create the path for the decompressed file. May contain regex
      capture groups, such as \1, \2, ...""")

  args, pipeline_args = parser.parse_known_args()
  beam_options = PipelineOptions(pipeline_args)
  # serialize and provide global imports, functions, etc. to workers.
  beam_options.view_as(SetupOptions).save_main_session = True
  beam_options.view_as(SetupOptions).setup_file = get_setup_file()

  with beam.Pipeline(options=beam_options) as p:
    _ = \
      (p
       | beam.Create(args.gcs_input_prefix)
       | 'GetPaths' >> beam.ParDo(GetPaths(validation_regex=args.path_sub_pattern))
       # Materialize and re-bundle paths with ReShuffle to enable parallelism.
       | beam.Reshuffle()
       | 'DecompressAndWrite' >> beam.ParDo(
           DecompressAndWrite(args.path_sub_pattern, args.path_sub_repl)))


if __name__ == '__main__':
  main()
