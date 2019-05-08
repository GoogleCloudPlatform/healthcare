r"""Untar .tar and .tar.gz GCS files."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import apache_beam as beam
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.options.pipeline_options import SetupOptions
from datathon_etl_pipelines.dofns.read_tar_file import ReadTarFile
from datathon_etl_pipelines.utils import get_setup_file
import tensorflow as tf


def write_file(element):
  path, contents = element
  with tf.io.gfile.GFile(path, 'wb') as fp:
    fp.write(contents)


def main():
  """Build and execute the Apache Beam pipeline using the commandline arguments."""
  parser = argparse.ArgumentParser(description=__doc__)
  parser.add_argument(
      '--input_tars',
      required=True,
      nargs='+',
      help="""One or more wildcard patterns that give the full paths to the
      input tar files on GCS.""")

  parser.add_argument(
      '--output_dir',
      required=True,
      help="""The output directory to write the untar'd files to.""")

  args, pipeline_args = parser.parse_known_args()
  beam_options = PipelineOptions(pipeline_args)
  # serialize and provide global imports, functions, etc. to workers.
  beam_options.view_as(SetupOptions).save_main_session = True
  beam_options.view_as(SetupOptions).setup_file = get_setup_file()

  if args.output_dir.endswith('/'):
    out_dir = args.output_dir[:-1]
  else:
    out_dir = args.output_dir

  def get_full_output_path(relative_path):
    if relative_path.startswith('/'):
      return out_dir + relative_path
    else:
      return '{}/{}'.format(out_dir, relative_path)

  with beam.Pipeline(options=beam_options) as p:
    _ = \
      (p
       | beam.Create(tf.io.gfile.glob(args.input_tars))
       | 'Untar' >> beam.ParDo(ReadTarFile(), get_full_output_path)
       | 'Write' >> beam.Map(write_file))


if __name__ == '__main__':
  main()
