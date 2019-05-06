r"""Load the MIMIC CXR dataset onto GCP.

The raw MIMIC CXR dataset is originally hosted by Physionet as a collection of
tar files which contain JPG images, and a gzip'd CSV file of labels.

The JPG images have paths of the form:
(train|valid)/p([0-9]+)/s([0-9]+)/view([0-9]+)_(frontal|lateral|other)\.jpg
which lists the dataset, patient id, study id, image number and view for each
image.

The gzip'd CSV file contains columns:
  - path
  - view
  - No Finding
  - Enlarged Cardiomediastinum
  - Cardiomegaly
  - Airspace Opacity
  - Lung Lesion
  - Edema
  - Consolidation
  - Pneumonia
  - Atelectasis
  - Pneumothorax
  - Pleural Effusion
  - Pleural Other
  - Fracture
  - Support Devices

See https://physionet.org/works/MIMICCXR/files/ for more details and to download
this data.

This apache beam pipeline takes this set of files as input, and outputs a
BigQuery table, two TFRecords (one for frontal and one for lateral images),
and the untar'd jpg images. This delivers an ergonomic presentation of the
dataset that enables high productivity for data scientists working with this
data at datathons.

All the provided paths must be absolute and may point to either local files
or GCS blobs.
"""
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import csv
import inspect
import os
import re

import apache_beam as beam
from apache_beam.io.gcp.bigquery import BigQueryDisposition
from apache_beam.io.gcp.bigquery import WriteToBigQuery
from apache_beam.io.gcp.bigquery_tools import parse_table_schema_from_json
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.options.pipeline_options import SetupOptions
from datathon_etl_pipelines.dofns.read_tar_file import ReadTarFile
from datathon_etl_pipelines.dofns.resize_image import ResizeImage
from datathon_etl_pipelines.mimic_cxr.enum_encodings import DATASET_VALUES
from datathon_etl_pipelines.mimic_cxr.enum_encodings import ID_NAMES
from datathon_etl_pipelines.mimic_cxr.enum_encodings import LABEL_NAMES
from datathon_etl_pipelines.mimic_cxr.enum_encodings import LABEL_VALUES
from datathon_etl_pipelines.mimic_cxr.enum_encodings import VIEW_VALUES
import tensorflow as tf
from typing import Dict, Union


def path_to_ids(path):
  """Parse the path to an image in the dataset.

  Args:
    path (str): A relative path to a JPG image.

  Returns:
    Tuple[int, int, int, int, int]: the patient_id, study_id, image_id, view,
    and dataset for this image.
  """
  match = re.match(
      r'(train|valid)/p([0-9]+)/s([0-9]+)/view([0-9]+)'
      r'_(frontal|lateral|other)\.jpg', path)
  if match is not None:
    d, pid, sid, iid, v = match.groups()
    patient_id = int(pid)
    study_id = int(sid)
    image_id = int(iid)
    view = VIEW_VALUES[v]
    dataset = DATASET_VALUES[d]
    return patient_id, study_id, image_id, view, dataset
  raise ValueError('unrecognized path: {}'.format(path))


def ids_to_path(ids):
  id_dict = dict(zip(ID_NAMES, ids))  # type: Dict[str, Union[int, str]]
  id_dict['view'] = VIEW_VALUES(id_dict['view'])
  id_dict['dataset'] = DATASET_VALUES(id_dict['dataset'])
  return '{dataset}/p{patient}/s{study:02d}/view{image}_{view}.jpg'.format(
      **id_dict)


class ChexpertConverter(object):
  """Converts CheXpert codes into integers mapped by LABEL_VALUES."""
  chexpert_code_to_index = {
      '': LABEL_VALUES['not_mentioned'],
      '1.0': LABEL_VALUES['positive'],
      '-1.0': LABEL_VALUES['uncertain'],
      '0.0': LABEL_VALUES['negative']
  }

  @classmethod
  def convert(cls, s):
    int_code = cls.chexpert_code_to_index.get(s, None)
    if int_code is None:
      raise ValueError('unrecognized chexpert encoding: {}'.format(s))
    else:
      return int_code


def parse_csv_row(line):
  row = next(csv.reader([line]))
  ids = path_to_ids(row[0])
  # row[1] is the view, which is already captured within the ids
  return ids, [ChexpertConverter.convert(s) for s in row[2:]]


def to_bigquery_json(ids_row):
  ids, row = ids_row
  bq = {col: val for col, val in zip(LABEL_NAMES, row)}
  bq.update(**dict(zip(ID_NAMES, ids)))
  del bq['dataset']
  bq['path'] = ids_to_path(ids)
  return bq


def write_jpg(element, path):
  ids, jpg_bytes = element
  file_path = path + ids_to_path(ids)
  tf.io.gfile.makedirs(file_path.rsplit('/', 1)[0])
  with tf.io.gfile.GFile(file_path, 'wb') as gf:
    gf.write(jpg_bytes)


def to_tf_example(element):
  """Construct a tf.train.Example from an image joined with labels.

  Args:
    element (Tuple[Tuple, Dict[Union['jpgs', 'rows'], List[List[int]]]]): The
      set of ids for the image along with the joined image and labels.

  Returns:
    tf.train.Example: The labelled image.
  """
  (ids, join) = element
  (patient, study, image, view, _) = ids
  if len(join['jpgs']) != 1 or len(join['rows']) != 1:
    raise ValueError('{} JPG files matched to {} rows for {}'.format(
        len(join['jpgs']), len(join['rows']), ids_to_path(ids)))

  def _bytes_feature(v):
    return tf.train.Feature(bytes_list=tf.train.BytesList(value=[v]))

  def _int64_feature(v):
    return tf.train.Feature(int64_list=tf.train.Int64List(value=[v]))

  features = {
      'jpg_bytes': _bytes_feature(join['jpgs'][0]),
      'patient': _int64_feature(patient),
      'study': _int64_feature(study),
      'image': _int64_feature(image),
      'view': _int64_feature(view),
  }

  for label, value in zip(LABEL_NAMES, join['rows'][0]):
    features[label] = _int64_feature(value)

  example = tf.train.Example(features=tf.train.Features(feature=features))
  return example


def import_json_bq_schema():
  path = os.path.join(
      os.path.dirname(inspect.getfile(inspect.currentframe())),
      'mimic_cxr_bq_schema.json')

  with open(path) as fp:
    return parse_table_schema_from_json(fp.read())


def build_and_run_pipeline(pipeline_options,
                           input_tars,
                           input_csv,
                           output_jpg_dir,
                           output_bq_table,
                           output_tfrecord_dir,
                           output_image_shape=None):
  """Construct and run the Apache Beam Pipeline.

  Args:
    pipeline_options (PipelineOptions): The passed to Apache Beam.
    input_tars (List[str]): A set of patterns specifying the paths to the input
      tar files.
    input_csv (str): The path to the (optionally compressed) CSV that contains
      the image labels.
    output_jpg_dir (str): The directory to output the JPG files to.
    output_bq_table (str): A string of the form `project:dataset.table_name`.
      This table will be overwritten if it already exists.
    output_tfrecord_dir (str): The directory to output the sharded TFRecords to.
    output_image_shape (Optional[Tuple]): The dimensions to resize the image to.
      Either HW or HWC. If this is None, then the images will not be resized.
  """
  input_paths = []
  for pattern in input_tars:
    input_paths.extend(tf.gfile.Glob(pattern))

  if not input_paths:
    raise ValueError('No matching tar files were found.')

  with beam.Pipeline(options=pipeline_options) as p:

    rows = (
        p
        | beam.io.ReadFromText(input_csv, skip_header_lines=1)
        | beam.Map(parse_csv_row))

    if output_bq_table is not None:
      _ = rows | beam.Map(to_bigquery_json) | WriteToBigQuery(
          table=output_bq_table,
          schema=import_json_bq_schema(),
          write_disposition=BigQueryDisposition.WRITE_APPEND)

    if output_jpg_dir is not None or output_tfrecord_dir is not None:
      jpgs = p | beam.Create(input_paths) | beam.ParDo(ReadTarFile(),
                                                       path_to_ids)

      if output_image_shape is not None:
        h = output_image_shape[0]
        w = output_image_shape[1]
        c = None
        if len(output_image_shape) > 2:
          c = output_image_shape[2]
        jpgs |= beam.ParDo(ResizeImage(h, w, c, 'jpg'))

      if output_jpg_dir is not None:
        if not output_jpg_dir.endswith('/'):
          output_jpg_dir += '/'
        _ = jpgs | beam.Map(write_jpg, output_jpg_dir)

      joined = {'jpgs': jpgs, 'rows': rows} | beam.CoGroupByKey()

      frontal, lateral, _ = joined | beam.Partition(
          lambda e, n: e[0][ID_NAMES['view']], 3)

      if output_tfrecord_dir is not None:
        if not output_tfrecord_dir.endswith('/'):
          output_tfrecord_dir += '/'
        for pcol, name in [(frontal, 'frontal'), (lateral, 'lateral')]:
          _ = (
              pcol
              | (name + '_to_tf_example') >> beam.Map(to_tf_example)
              | (name + '_write_tf_record') >> beam.io.WriteToTFRecord(
                  output_tfrecord_dir + name,
                  file_name_suffix='.tfrecord',
                  coder=beam.coders.ProtoCoder(tf.train.Example)))


def get_setup_file():
  this_file = inspect.getfile(inspect.currentframe())
  this_dir = os.path.dirname(this_file)
  relative_path = os.path.join(this_dir, '..', '..', 'setup.py')
  return os.path.abspath(relative_path)


def main():
  parser = argparse.ArgumentParser(description=__doc__)
  parser.add_argument(
      '--input_tars',
      nargs='+',
      required=True,
      help='A set of patterns specifying the paths to the input tar files.')
  parser.add_argument(
      '--input_csv',
      required=True,
      help='The path to the (optionally compressed) CSV that contains the image'
      ' labels.')
  parser.add_argument(
      '--output_jpg_dir',
      required=False,
      help='The directory to output the JPG files to.')
  parser.add_argument(
      '--output_bq_table',
      required=False,
      help='A string of the form `project:dataset.table_name`. This table will '
      'be overwritten if it already exists.')
  parser.add_argument(
      '--output_tfrecord_dir',
      required=False,
      help='The directory to output the sharded TFRecords to.')
  parser.add_argument(
      '--output_image_shape',
      nargs='+',
      type=int,
      required=False,
      help='The dimensions to resize the image to. Either HW or HWC. If this is'
      ' None, then the images will not be resized.')

  args, pipeline_args = parser.parse_known_args()
  beam_options = PipelineOptions(pipeline_args)
  beam_options.view_as(SetupOptions).save_main_session = True
  beam_options.view_as(SetupOptions).setup_file = get_setup_file()

  if args.output_image_shape is not None:
    if len(args.output_image_shape) not in (2, 3):
      parser.error('2 (HW) or 3 (HWC) integers are required for '
                   'output_image_shape')

  build_and_run_pipeline(
      pipeline_options=beam_options,
      input_tars=args.input_tars,
      input_csv=args.input_csv,
      output_jpg_dir=args.output_jpg_dir,
      output_bq_table=args.output_bq_table,
      output_tfrecord_dir=args.output_tfrecord_dir,
      output_image_shape=args.output_image_shape)


if __name__ == '__main__':
  main()
