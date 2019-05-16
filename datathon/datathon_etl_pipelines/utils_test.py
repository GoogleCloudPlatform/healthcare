import decimal
import os

from absl.testing import absltest
from absl.testing import parameterized
from datathon_etl_pipelines.utils import get_setup_file
from datathon_etl_pipelines.utils import scalar_to_feature
from enum import Enum


class GetSetupFileTest(absltest.TestCase):

  def test_exists(self):
    self.assertTrue(os.path.isfile(get_setup_file()))

  def test_exists_diff_directory(self):
    os.chdir('/')
    self.assertTrue(os.path.isfile(get_setup_file()))


class DummyEnum(Enum):
  a = 0


class AnyToFeatureTest(parameterized.TestCase):

  @parameterized.named_parameters(('int', 3), ('bool', False))
  def test_int_conversions(self, value):
    example = scalar_to_feature(value)
    self.assertNotEmpty(example.int64_list.value)
    self.assertEmpty(example.float_list.value)
    self.assertEmpty(example.bytes_list.value)

  @parameterized.named_parameters(('float', 13.3),
                                  ('decimal', decimal.Decimal('0.3')))
  def test_float_conversions(self, value):
    example = scalar_to_feature(value)
    self.assertEmpty(example.int64_list.value)
    self.assertNotEmpty(example.float_list.value)
    self.assertEmpty(example.bytes_list.value)

  @parameterized.named_parameters(('unicode', u'abc'), ('bytes', b'123'),
                                  ('string', 'something'))
  def test_string_conversions(self, value):
    example = scalar_to_feature(value)
    self.assertEmpty(example.int64_list.value)
    self.assertEmpty(example.float_list.value)
    self.assertNotEmpty(example.bytes_list.value)


if __name__ == '__main__':
  absltest.main()
