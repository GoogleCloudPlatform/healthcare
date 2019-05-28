import os

from absl.testing import absltest
from absl.testing import parameterized
from datathon_etl_pipelines.generic_imaging.inference_to_bigquery import default_normalize
from datathon_etl_pipelines.generic_imaging.inference_to_bigquery import ExampleWithImageBytesToInput
from datathon_etl_pipelines.utils import bytes_feature
from datathon_etl_pipelines.utils import get_test_data
from datathon_etl_pipelines.utils import int64_feature
import numpy as np
import tensorflow as tf


def custom_normalize(image):
  return (1 / 127.5) * (tf.cast(image, tf.float32) - 127.5)


class ExampleWithImageBytesToInputTest(parameterized.TestCase):

  @parameterized.named_parameters(
      ('jpg', 'jpg', default_normalize, (0, 1)),
      ('png', 'png', default_normalize, (0, 1)),
      ('jpg_and_other_features', 'png', default_normalize, (0, 1)),
      ('jpg_custom_fn', 'png', custom_normalize, (-1, 1)),
  )
  def test_mandrill_bytes_to_image(self, image_format, image_process_fn,
                                   bounds):
    features = dict()
    image_path = '{}.{}'.format(
        os.path.join(get_test_data(), 'mandrill'), image_format)
    with open(image_path, 'rb') as f:
      features[image_format + '_bytes'] = bytes_feature(f.read())
    example = tf.train.Example(features=tf.train.Features(feature=features))

    fn = ExampleWithImageBytesToInput(
        image_format=image_format, image_process_fn=image_process_fn)

    result = fn(example)

    self.assertTrue(np.all(bounds[0] <= result))
    self.assertTrue(np.all(result <= bounds[1]))
    self.assertEqual(result.shape, (512, 512, 3))

  @parameterized.named_parameters(
      ('jpg', 'jpg', default_normalize, (0, 1)),
      ('png', 'png', default_normalize, (0, 1)),
      ('jpg_and_other_features', 'png', default_normalize, (0, 1)),
      ('jpg_custom_fn', 'png', custom_normalize, (-1, 1)))
  def test_mandrill_bytes_to_image_with_multiple_features(
      self, image_format, image_process_fn, bounds):
    features = dict()
    image_path = '{}.{}'.format(
        os.path.join(get_test_data(), 'mandrill'), image_format)
    with open(image_path, 'rb') as f:
      features[image_format + '_bytes'] = bytes_feature(f.read())
    features['some_other_feature'] = int64_feature(3)
    features['feature_bytes'] = bytes_feature(b'test-a')
    features['bytes_feature'] = bytes_feature(b'test-b')
    example = tf.train.Example(features=tf.train.Features(feature=features))
    fn = ExampleWithImageBytesToInput(
        image_format=image_format, image_process_fn=image_process_fn)

    result = fn(example)

    self.assertTrue(np.all(bounds[0] <= result))
    self.assertTrue(np.all(result <= bounds[1]))
    self.assertEqual(result.shape, (512, 512, 3))


if __name__ == '__main__':
  absltest.main()
