import argparse
import apache_beam as beam

parser = argparse.ArgumentParser()
parser.add_argument('--start-pipeline-path', default= 'gs://PREFIX-ENVIRONMENT-csv-input/CSV_DATA_SOURCE/CSV_DATA_SOURCE_{TIMESTAMP}/START_PIPELINE')


def udf():
  file = io.gcsio.GcsIO().open(filename =gcs_path, mode = 'w', mime_type='application/csv')
  yield

with beam.Pipeline(options=beam_options) as pipeline:
  p = (
      pipeline | beam.ParDo(udf())
  )