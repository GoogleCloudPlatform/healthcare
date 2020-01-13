# Copyright 2018 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Utility functions for training and serving ML models."""

import warnings

# TODO: Remove when Tensorflow library is updated.
warnings.filterwarnings('ignore')
import tensorflow.compat.v1 as tf
import tensorflow_hub

# URL to the trained neural net, which gets feature vectors from images. It is a
# checkpoint of Inception V3 model trained on ImageNet with the few last
# classification layers stripped.
_FEATURE_VECTORS_MODULE_URL = 'https://tfhub.dev/google/imagenet/inception_v3/feature_vector/1'


def get_bottleneck_tensor(input_jpeg_str):
  # type: tf.Tensor -> tf.Tensor
  """Calculates the bottleneck tensor for input JPEG string tensor.

  This function will resize/encode the image as required by Inception V3 model.
  Then it will run it through the InceptionV3 checkpoint to calculate
  bottleneck values.

  Args:
    input_jpeg_str: Tensor for input JPEG image.

  Returns:
    bottleneck_tensor: Tensor for output bottleneck Tensor.
  """
  module_spec = tensorflow_hub.load_module_spec(_FEATURE_VECTORS_MODULE_URL)
  input_height, input_width = tensorflow_hub.get_expected_image_size(
      module_spec)
  input_depth = tensorflow_hub.get_num_image_channels(module_spec)
  decoded_image = tf.image.decode_jpeg(input_jpeg_str, channels=input_depth)
  decoded_image_as_float = tf.image.convert_image_dtype(decoded_image,
                                                        tf.float32)
  decoded_image_4d = tf.expand_dims(decoded_image_as_float, 0)
  resize_shape = tf.stack([input_height, input_width])
  resize_shape_as_int = tf.cast(resize_shape, dtype=tf.int32)
  resized_image_4d = tf.image.resize_bilinear(decoded_image_4d,
                                              resize_shape_as_int)
  m = tensorflow_hub.Module(module_spec)
  bottleneck_tensor = m(resized_image_4d)
  return bottleneck_tensor
