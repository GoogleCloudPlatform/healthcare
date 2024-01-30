import argparse

import apache_beam as beam
from apache_beam.options.pipeline_options import PipelineOptions

parser = argparse.ArgumentParser()
parser.add_argument('--input-csv', default='gs://input-bucket-here')
args, beam_args = parser.parse_known_args()

# Create and set your Pipeline Options.
beam_options = PipelineOptions(beam_args)


def udf():
    # Define custom CSV transformation logic here
    yield

with beam.Pipeline(options=beam_options) as pipeline:
  p = (
      pipeline
      | beam.io.ReadFromText(args.input)
      | beam.ParDo(udf())
      | beam.io.WriteToText(args.input)
  )

# Note that the final WriteToText transformation in the pipeline would require Storage.object.delete permissions
# in the pipeline on GCP, which may not be granted for business reasons. If this is the case, please write to
# GCS using the API provided here:
# https://github.com/googleapis/python-storage/blob/HEAD/samples/snippets/storage_fileio_write_read.py`)