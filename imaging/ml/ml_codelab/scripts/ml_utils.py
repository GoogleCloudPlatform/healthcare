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
import scripts.constants as constants

# TODO: Remove when Tensorflow library is updated.
warnings.filterwarnings('ignore')
import tensorflow.compat.v1 as tf
import tensorflow_hub

# URL to the trained neural net, which gets feature vectors from images. It is a
# checkpoint of Inception V3 model trained on ImageNet with the few last
# classification layers stripped.
_FEATURE_VECTORS_MODULE_URL = 'https://tfhub.dev/google/imagenet/inception_v3/feature_vector/1'


def create_fixed_weight_input_graph(
    input_pixels_tensor_name=constants.INPUT_PIXELS_TENSOR_NAME):
  # type: str -> (tf.Tensor, tf.Tensor)
  """Creates the input graph (with fixed weights) using TF Hub.

  This function will create the input graph consisting of nodes that are
  used to resize/encode the image followed by the nodes of the Inception V3
  model. The classification head is not added in this function.

  Args:
    input_pixels_tensor_name: Name of input tensor.

  Returns:
    img_bytes: Tensor for input bytes.
    bottleneck_tensor: Tensor for output bottleneck.
  """

  def decode_and_resize_img(img_bytes, height, width, depth):
    # type: (tf.Tensor, int, int, int) -> tf.Tensor
    """Decodes and resizes input image to return a float representation.

    Args:
      img_bytes: tensor for input bytes.
      height: Desired height of image.
      width: Desired width of image.
      depth: Desired bit depth of image.

    Returns:
      float_pixels: Tensor storing the image as float. This is the input tensor
      that we'll reference in the Explainable AI feature to show how output
      changes with input.
    """
    features = tf.squeeze(img_bytes, axis=1, name='input_squeeze')
    float_pixels = tf.map_fn(
        # pylint: disable=g-long-lambda
        lambda img: tf.image.resize_with_crop_or_pad(
            tf.io.decode_image(img, channels=depth, dtype=tf.float32), height,
            width),
        features,
        dtype=tf.float32,
        name='input_convert')
    float_pixels = tf.ensure_shape(float_pixels, (None, height, width, depth))
    float_pixels = tf.identity(float_pixels, name=input_pixels_tensor_name)
    return float_pixels

  def download_image_model(mdl_url):
    # type: str -> (tensorflow_hub.Module, int, int, int)
    """Returns the Tensorflow Hub model used to process images."""
    module_spec = tensorflow_hub.load_module_spec(mdl_url)
    input_height, input_width = tensorflow_hub.get_expected_image_size(
        module_spec)
    input_depth = tensorflow_hub.get_num_image_channels(module_spec)
    m = tensorflow_hub.Module(module_spec)
    return (m, input_height, input_width, input_depth)

  m, height, width, color_depth = download_image_model(
      _FEATURE_VECTORS_MODULE_URL)
  img_bytes = tf.placeholder(tf.string, shape=(None, 1))
  img_float = decode_and_resize_img(img_bytes, height, width, color_depth)
  bottleneck_tensor = m(img_float)
  return img_bytes, bottleneck_tensor
