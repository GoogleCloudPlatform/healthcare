"""Prepare DICOM data for a Datathon.

If given a collection of .dcm files in Google Cloud Storage, this script
bulk imports them into a Google Cloud Healthcare DICOM Store.

After that, this script will optionally export the DICOM metadata in that
DICOM Store into a BigQuery Table, and optionally export the image data to
.png files stored in Google Cloud Storage.

All the required resources (Cloud Healthcare Dataset, Cloud Healthcare
DICOM Store, BigQuery Dataset, BigQuery Table and Google Cloud Storage
bucket(s)) are created if they do not exist.

If the chosen Cloud Healthcare DICOM Store already exists, any new instances
(as determined by the study, series and instance UIDs) provided via
--gcs_dicom_pattern will be added to the DICOM Store, and the entire DICOM Store
will be exported to BigQuery and GCS .png files.

The .png files output by this script may be converted into TFRecord files using
`png_to_tfrecord.py`.
"""
import argparse
import logging
import re
import cloud.healthcare as chc
import concurrent.futures
import google.api_core.exceptions
from google.cloud import bigquery
from google.cloud import storage

logger = logging.getLogger('progress')
logger.addHandler(logging.StreamHandler())
logger.setLevel(logging.INFO)


def main(args):
  client = chc.Client()
  # Create CHC dataset if necessary
  try:
    dataset = client.create_dataset(args.chc_dataset)
    logger.info('Created Cloud Healthcare dataset: %s', args.chc_dataset)
  except google.cloud.exceptions.Conflict:
    # the CHC dataset was already created
    dataset = client.dataset(args.chc_dataset)

  # Create Cloud Healthcare DICOM Store if necessary
  try:
    dicom_store = dataset.create_dicom_store(args.chc_dicom_store)
    logger.info('Created Cloud Healthcare DICOM Store: %s', args.chc_dataset)
  except google.cloud.exceptions.Conflict:
    # the DICOM Store was already created
    dicom_store = dataset.dicom_store(args.chc_dicom_store)

  # Perform import
  # GCS Dicom -> Cloud Healthcare DICOM Store
  if args.gcs_dicom_pattern:
    logger.info('Importing dicom files into Google Cloud Healthcare')
    job = dicom_store.import_from_gcs(args.gcs_dicom_pattern)
    # wait on future with job.exception
    if job.exception() is not None:
      raise job.exception()
    logger.info('Successfully imported dicom'
                'files into Google Cloud Healthcare!')

  # Perform exports asynchronously
  export_jobs = dict()
  polling_executor = concurrent.futures.ThreadPoolExecutor(max_workers=2)
  # Cloud Healthcare DICOM Store -> BigQuery
  if args.bq_dataset is not None:
    # Create the BigQuery Dataset if necessary
    bq_client = bigquery.Client()
    try:
      bq_client.dataset(args.bq_dataset).create()
      logging.info('Created BigQuery dataset: %s', args.bq_dataset)
    except google.cloud.exceptions.Conflict:
      # Dataset already exists
      pass
    logger.info('Exporting dicom metadata to BigQuery')
    job = dicom_store.export_to_bigquery(args.bq_dataset, args.bq_table,
                                         args.overwrite_table)
    export_jobs[polling_executor.submit(polling_task, job)] = 'BigQuery'

  # Cloud Healthcare DICOM Store -> png
  if args.gcs_png_path is not None:
    # Create the GCS Bucket if it doesn't exist
    gcs_client = storage.Client()
    bucket, = re.match(r'gs://([a-z\-_.]+)/', args.gcs_png_path).groups()
    try:
      gcs_client.create_bucket(bucket)
      logging.info('Created GCS bucket: %s', bucket)
    except google.cloud.exceptions.Conflict:
      # Bucket already exists
      pass
    logger.info('Exporting dicom pixel data'
                ' to .png images located in %s', args.gcs_png_path)
    job = dicom_store.export_to_gcs(args.gcs_png_path, 'image/png')
    export_jobs[polling_executor.submit(polling_task, job)] = 'png'

  for future in concurrent.futures.as_completed(export_jobs):
    if future.exception() is None:
      if export_jobs[future] == 'BigQuery':
        logger.info('Successfully exported dicom metadata into BigQuery!')
      elif export_jobs[future] == 'png':
        logger.info('Successfully exported .png images!')

  for future, job_name in export_jobs.items():
    if future.exception() is not None:
      logger.error('%s export failed: %s', job_name, future.exception())


def polling_task(job):
  """Wraps an polling future so its polling can be done by an python thread."""
  if job.exception() is not None:
    raise job.exception()
  else:
    return job.result()


if __name__ == '__main__':
  parser = argparse.ArgumentParser(description=__doc__)
  parser.add_argument(
      '--gcs_dicom_pattern',
      required=False,
      help="""A Google Cloud Storage URI containing  the dicom files to import
      into Google Cloud  Healthcare. If this is not specified, then no
      additional DICOM instances will be added to the  DICOM Store before
      exporting. The URI, if provided, must be in the following format:
      gs://bucket-id/object-id. The object-id may include the directory
      separator: / and wildcards in object-id: **, *, ?""")

  parser.add_argument(
      '--chc_dataset',
      required=True,
      help="""The name of the Cloud Healthcare dataset that holds/will hold the
      DICOM Store.""")

  parser.add_argument(
      '--chc_dicom_store',
      required=True,
      help='The DICOM Store to import into and/or export from.')

  parser.add_argument(
      '--bq_dataset',
      required=False,
      help="""The name of the BigQuery dataset to store the exported DICOM
      metadata. If this and --bq_table are not specified, then the metadata will
      will not be exported to BigQuery.""")

  parser.add_argument(
      '--bq_table',
      required=False,
      help="""The name of the BigQuery table that will be created to hold the
      metadata for all of the DICOM instances in the given Cloud Healthcare
      DICOM Store.""")

  parser.add_argument(
      '--overwrite_table',
      action='store_true',
      required=False,
      help="""If this option is set, and the requested BigQuery dataset and
      table already exist, they will be silently overwritten. Otherwise,
      attempting to overwrite a BigQuery table will result in an error.""")

  parser.add_argument(
      '--gcs_png_path',
      required=False,
      help="""The GCS URI prefix to use for the exported .png image files. If
      this argument is not supplied then the DICOM instances will not be
      exported to .png format. This is of the form gs://{bucket}[/prefix/path].
      A trailing slash will be appended if it is not supplied. Output png files
      will have URIs:
      {gcs_png_path}/{instance_uid}/{series_uid}/{instance_uid}.png""")

  cl_args = parser.parse_args()
  # Validate the arguments.
  # Check if full BigQuery specification is given
  if bool(cl_args.bq_dataset) ^ bool(cl_args.bq_table):
    parser.error('--bq_dataset and --bq_table must be '
                 ' provided together or not at all.')
  # Supplying useless arguments signals user misunderstanding and is
  # treated as an error.
  if cl_args.overwrite_table and not cl_args.bq_dataset:
    parser.error('--overwrite_table should only be specified if BigQuery '
                 'export is requested as part of the execution of this script. '
                 'To request a BigQuery export, provide --bq_dataset and '
                 '--bq_table arguments.')

  # Check if the user input matches the form of a valid GCS URI
  if cl_args.gcs_dicom_pattern is not None and \
    not re.match(r'gs://[a-z0-9\-_.]+/.+', cl_args.gcs_dicom_pattern):
    parser.error('Invalid argument for --gcs_dicom_pattern: {}'.format(
        cl_args.gcs_dicom_pattern))
  if cl_args.gcs_png_path is not None and \
      not re.match(r'gs://[a-z0-9\-_.]+', cl_args.gcs_png_path):
    parser.error('Invalid argument for --gcs_png_path: {}'.format(
        cl_args.gcs_png_path))
  main(cl_args)
