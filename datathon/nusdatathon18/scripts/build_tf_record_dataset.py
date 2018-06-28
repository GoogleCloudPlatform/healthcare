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

r"""
Convert existing images to TFRecords
(https://www.tensorflow.org/programmers_guide/datasets).

Example usage:

python build_tf_record_dataset.py --src_bucket datathon-cbis-ddsm-colab \
                                  --src_folder train \
                                  --dst_bucket datathon-cbis-ddsm-colab \
                                  --dst_file cache/train.tfrecords
"""

import argparse
import numpy as np
import tensorflow as tf

from google.cloud import storage
from io import BytesIO
from PIL import Image

client = storage.Client()

def load_data(bucket, folder):
  """
  Loads images from a GCS bucket, and parses labels from filenames.

  Args:
    bucket: A GCS bucket.
    folder: A subfolder inside the bucket.

  Returns:
    An array of images as numpy matrices, and an array of labels.
  """
  images = []
  labels = []
  # 1, 2, 3 and 4 are breast density categories.
  for label in [1, 2, 3, 4]:
    blobs = bucket.list_blobs(prefix=("%s/%s_" % (folder, label)))

    for blob in blobs:
      byte_stream = BytesIO()
      blob.download_to_file(byte_stream)
      byte_stream.seek(0)

      img = Image.open(byte_stream)
      images.append(np.array(img, dtype=np.float32))
      labels.append(label-1) # Minus 1 to fit in [0, 4) for evaluation.

  return images, labels

def load_and_save_as_tf_records(src_bucket, src_folder, dst_bucket, dst_file):
  """Converts images and labels to TFRecords."""
  images, labels = load_data(src_bucket, src_folder)

  record_path = "gs://%s/%s" % (dst_bucket.name, dst_file)
  with tf.python_io.TFRecordWriter(record_path) as w:
    for i, image in enumerate(images):
      feature = {
        "label": tf.train.Feature(
          int64_list=tf.train.Int64List(value=[labels[i]])),
        "image": tf.train.Feature(
          bytes_list=tf.train.BytesList(
            value=[tf.compat.as_bytes(image.tostring())])),
      }
      example = tf.train.Example(features=tf.train.Features(feature=feature))
      w.write(example.SerializeToString())

def run(args):
  src_bucket = client.get_bucket(args.src_bucket)
  dst_bucket = client.get_bucket(args.dst_bucket)
  load_and_save_as_tf_records(
    src_bucket, args.src_folder, dst_bucket, args.dst_file)

if __name__ == '__main__':
  parser = argparse.ArgumentParser(description='Transform to TFRecords.')
  parser.add_argument('--src_bucket', type=str, required=True,
    help='A GCS bucket to read source images.')
  parser.add_argument('--src_folder', type=str, required=True,
    help='A folder within the GCS bucket to read source images.')
  parser.add_argument('--dst_bucket', type=str, required=True,
    help='A GCS bucket to write generated TFRecords.')
  parser.add_argument('--dst_file', type=str, required=True,
    help='A file within the GCS bucket to write TFRecords.')
  args = parser.parse_args()

  run(args)