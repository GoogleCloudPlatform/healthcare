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
r"""Select train/test images for demo.

This script is used to select training and test images for the DDSM Machine
Learning colab (See the tutorials folder up one level) use.

Training images are selected based on the breast density, 25 images per
category. Test images are selected randomly, with a total count of 20.

Note that only "Cranial-Caudal" (https://breast-cancer.ca/mammopics/) images
are considered because we don't want mixed CC and MLO images.

Example usage:

python select_demo_images.py --src_bucket=datathon-cbis-ddsm-colab \
                             --src_training_folder=train \
                             --src_eval_folder=test \
                             --dst_bucket=datathon-cbis-ddsm-colab \
                             --dst_training_folder=train_demo \
                             --dst_eval_folder=test_demo \
                             --training_size_per_cat 25 \
                             --eval_size 20
"""

import argparse
import random
from google.cloud import storage

client = storage.Client()


def select_training_data(src_bucket, dst_bucket, args):
  # 1, 2, 3 and 4 are breast density categories.
  for i in [1, 2, 3, 4]:
    blobs = src_bucket.list_blobs(
        prefix=('%s/%s_' % (args.src_training_folder, i)))

    c = 0
    for blob in blobs:
      if blob.name.endswith('_CC'):
        src_bucket.copy_blob(
            blob,
            dst_bucket,
            new_name=(
                '%s/%s' % (args.dst_training_folder, blob.name.split('/')[1])))

        c += 1
        # Select the images for each category.
        if c == args.training_size_per_cat:
          break


def select_test_data(src_bucket, dst_bucket, args):
  blobs = src_bucket.list_blobs(prefix=(args.src_eval_folder))

  objects = []
  for blob in blobs:
    if blob.name.endswith('_CC'):
      objects.append(blob)

  random.shuffle(objects)

  # Select random test images.
  for i in range(args.eval_size):
    src_bucket.copy_blob(
        objects[i],
        dst_bucket,
        new_name=(
            '%s/%s' % (args.dst_eval_folder, objects[i].name.split('/')[1])))


def run(args):
  src_bucket = client.get_bucket(args.src_bucket)
  dst_bucket = client.get_bucket(args.dst_bucket)

  select_training_data(src_bucket, dst_bucket, args)
  select_test_data(src_bucket, dst_bucket, args)


if __name__ == '__main__':
  parser = argparse.ArgumentParser(description='Select images for demo.')
  parser.add_argument(
      '--src_bucket',
      type=str,
      required=True,
      help='GCS bucket where original images locate.')
  parser.add_argument(
      '--src_training_folder',
      type=str,
      required=True,
      help='GCS folder to read training images from.')
  parser.add_argument(
      '--src_eval_folder',
      type=str,
      required=True,
      help='GCS folder to read evaluation images from.')
  parser.add_argument(
      '--dst_bucket',
      type=str,
      required=True,
      help='GCS bucket where demo images locate.')
  parser.add_argument(
      '--dst_training_folder',
      type=str,
      required=True,
      help='GCS folder to write demo training images to.')
  parser.add_argument(
      '--dst_eval_folder',
      type=str,
      required=True,
      help='GCS folder to write demo evaluation images to.')
  parser.add_argument(
      '--training_size_per_cat',
      type=int,
      required=True,
      help='Number of images to select per category for training.')
  parser.add_argument(
      '--eval_size',
      type=int,
      required=True,
      help='Number of images to select for evaluation.')
  args = parser.parse_args()

  run(args)
