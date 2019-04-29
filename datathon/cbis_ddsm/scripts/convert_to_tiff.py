#!/usr/bin/python
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
r"""Preprocess the DICOM images.

Pixel arrays are read from DICOM and converted to PNGs, maxinum width and
height of all images are recorded to be used for padding later.

Example usage:

python convert_to_tiff.py \
    --src_bucket=datathon-cbis-ddsm-images \
    --dst_bucket=datathon-cbis-ddsm-colab \
    --dst_folder=train \
    --label_file=gs://datathon-cbis-ddsm-images/calc_case_description_train_set.csv

"""

import argparse
from io import BytesIO
import math
import multiprocessing
from multiprocessing import Pool
import urllib
import numpy as np
import pandas as pd
import PIL
import pydicom
from tensorflow.python.lib.io import file_io
from google.cloud import storage

# Columns to extract from the label files.
BREAST_DENSITY_COL = 'breast density'
IMAGE_FILE_PATH_COL = 'image file path'


def convert((args, df)):
  """Extracts images from DICOM files and stores in GCS.

  Args:
    args: the arguments passed to this script.
    df: pandas dataframe to process.

  Returns:
    A tuple of max width and height of all the processed images.
  """
  # Create a client per process.
  client = storage.Client()
  src_bucket = client.get_bucket(args.src_bucket)
  dst_bucket = client.get_bucket(args.dst_bucket)

  def _read_image(path):
    blob = src_bucket.blob(path)
    byte_stream = BytesIO()
    blob.download_to_file(byte_stream)
    byte_stream.seek(0)
    return byte_stream

  def _upload_image(filename, byte_stream):
    blob = dst_bucket.blob(filename)
    byte_stream.seek(0)
    blob.upload_from_file(byte_stream)

  # Record max height and width for current process.
  max_h, max_w = -1, -1

  cols = df[[BREAST_DENSITY_COL, IMAGE_FILE_PATH_COL]]
  for _, t in cols.iterrows():
    path = t[IMAGE_FILE_PATH_COL].strip()

    filename = '%s/%s_%s' % (args.dst_folder, t[BREAST_DENSITY_COL],
                             path.split('/')[0])
    if dst_bucket.blob(filename).exists:
      print 'Skipping converted file: %s' % filename
      continue

    img = _read_image(urllib.unquote(path))
    dcm = pydicom.dcmread(img)

    arr = dcm.pixel_array

    (height, width) = arr.shape
    max_h = max(height, max_h)
    max_w = max(width, max_w)

    byte_stream = BytesIO()
    PIL.Image.fromarray(arr).save(byte_stream, format='TIFF')

    _upload_image(filename, byte_stream)

  return max_h, max_w


def run(args):
  with file_io.FileIO(args.label_file, 'r') as f:
    df = pd.read_csv(f)

    # Distribute the load on each core of CPU.
    core_cnt = multiprocessing.cpu_count()
    pool = Pool(core_cnt)

    row_cnt = len(df)
    per_core = int(math.ceil(row_cnt * 1.0 / core_cnt))

    data_slices = [[args, df[per_core * i:min(per_core * (i + 1), row_cnt)]]
                   for i in range(core_cnt)]

    def max_dimen(c, d):
      return (max(c[0], d[0]), max(c[1], d[1]))

    print reduce(max_dimen, pool.map(convert, data_slices))


if __name__ == '__main__':
  parser = argparse.ArgumentParser(description='.')
  parser.add_argument(
      '--src_bucket',
      type=str,
      required=True,
      help='GCS bucket where original images locate.')
  parser.add_argument(
      '--dst_bucket',
      type=str,
      required=True,
      help='GCS bucket where transformed images locate.')
  parser.add_argument(
      '--dst_folder',
      type=str,
      required=True,
      help='GCS folder to save images to.')
  parser.add_argument(
      '--label_file',
      type=str,
      required=True,
      help='The label file, should have "breast density" and "image file path" '
      'columns. The label files for DDSM can be downloaded from CBIS-DDSM '
      'website.')
  args = parser.parse_args()

  run(args)
