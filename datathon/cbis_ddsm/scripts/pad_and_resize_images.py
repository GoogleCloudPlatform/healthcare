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
r"""Pads the images to the desired ratio with black pixels, and resize to the given resolution.

Example usage:

python pad_and_resize_images.py --target_width 5251 \
                                --target_height 7111 \
                                --final_width 95 \
                                --final_height 128 \
                                --src_bucket datathon-cbis-ddsm-colab \
                                --dst_bucket datathon-cbis-ddsm-colab \
                                --dst_folder small_train \
                                --src_folder train
"""

import argparse
from io import BytesIO
import numpy as np
from PIL import Image
from google.cloud import storage

client = storage.Client()


def pad_and_resize_image(src_bucket, dst_bucket, blob, args):
  byte_stream = BytesIO()
  blob.download_to_file(byte_stream)
  byte_stream.seek(0)

  # Pad images by adding black pixels to right and bottom.
  img = np.array(Image.open(byte_stream))
  mask = np.full((args.target_height, args.target_width), 0)
  mask[:img.shape[0], :img.shape[1]] = img

  # Resize images to the desired size.
  upload_byte_stream = BytesIO()
  # We have to use uint32 instead of uint16.
  (Image.fromarray(mask.astype('uint32')).resize((args.final_width,
                                                  args.final_height)).save(
                                                      upload_byte_stream,
                                                      format='PNG'))

  blob = dst_bucket.blob(
      ('%s/%s' % (args.dst_folder, blob.name.split('/', 1)[1])))
  upload_byte_stream.seek(0)
  blob.upload_from_file(upload_byte_stream)


def run(args):
  src_bucket = client.get_bucket(args.src_bucket)
  blobs = src_bucket.list_blobs(prefix=('%s/' % args.src_folder))

  dst_bucket = client.get_bucket(args.dst_bucket)
  for blob in blobs:
    if not blob.name.endswith('/'):
      pad_and_resize_image(src_bucket, dst_bucket, blob, args)


if __name__ == '__main__':
  parser = argparse.ArgumentParser(description='Pad and resize images.')
  parser.add_argument(
      '--target_width',
      type=int,
      required=True,
      help='Target width to pad the images to.')
  parser.add_argument(
      '--target_height',
      type=int,
      required=True,
      help='Target height to pad the images to.')
  parser.add_argument(
      '--final_width',
      type=int,
      required=True,
      help='Final width to resize the images to.')
  parser.add_argument(
      '--final_height',
      type=int,
      required=True,
      help='Final height to resize the images to.')
  parser.add_argument(
      '--src_bucket',
      type=str,
      required=True,
      help='GCS bucket where original images locate.')
  parser.add_argument(
      '--src_folder',
      type=str,
      required=True,
      help='GCS folder to read images from.')
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
  args = parser.parse_args()

  run(args)
