import os

from absl.testing import absltest
from absl.testing import parameterized
from datathon_etl_pipelines.dofns.resize_image import ResizeImage
from datathon_etl_pipelines.utils import get_test_data
import tensorflow as tf


class ResizeColorImageTest(parameterized.TestCase):

  @parameterized.named_parameters(('jpg', 'jpg', (320, 320, 3)),
                                  ('png', 'png', (233, 233, 3)),
                                  ('jpg_bw', 'jpg', (61, 61, 1)))
  def test_mandrill(self, image_format, image_shape):
    dofn = ResizeImage(image_format, *image_shape)
    image_path = '{}.{}'.format(
        os.path.join(get_test_data(), 'mandrill'), image_format)
    with open(image_path, 'rb') as f:
      image_bytes = f.read()
    [(key, resized_image_bytes)] = list(dofn.process(('key', image_bytes)))
    self.assertEqual(key, 'key')
    resized_image_tensor = tf.io.decode_image(resized_image_bytes)
    with tf.Session() as session:
      resized_image = session.run([resized_image_tensor])
    self.assertEqual(resized_image.shape, image_shape)


if __name__ == '__main__':
  absltest.main()
