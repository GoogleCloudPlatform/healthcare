"""Tests for MLBQSharedType and MLBQConverter."""
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl.testing import absltest
from absl.testing import parameterized
from load_feature_vectors import MLBQConverter
from load_feature_vectors import MLBQSharedType
import numpy as np


class SharedTypeTest(parameterized.TestCase):
  _ans_nd = 'of_0_0:INT64,of_0_1:INT64,of_1_0:INT64,of_1_1:INT64'
  _ans_nd_mixed_float = _ans_nd.replace('INT64', 'FLOAT64')
  _ans_nd_mixed_string = _ans_nd.replace('INT64', 'STRING')
  _ans_long = ','.join('times_{:03}:INT64'.format(x) for x in xrange(101))

  @parameterized.named_parameters(
      ('string', 'it', 'a', 'it:STRING'),
      ('unicode', 'was', u'a', 'was:STRING'),
      ('float', 'the', 0.5, 'the:FLOAT64'), ('int', 'best', 1, 'best:INT64'),
      ('nD', 'of', [[1, 0], [0, 1]], _ans_nd),
      ('nD_mixed_float', 'of', [[1, 0.0], [0, 1]], _ans_nd_mixed_float),
      ('nD_mixed_string', 'of', [[1, 0.0], ['0', 1]], _ans_nd_mixed_string),
      ('long', 'times', range(101), _ans_long))
  def test_bq_string(self, name, obj, ans):
    shared_type = MLBQSharedType.from_object(obj)
    bq_string = shared_type.to_bigquery_schema(name)
    self.assertEqual(bq_string, ans)

  def test_from_object_empty(self):
    with self.assertRaises(ValueError):
      _ = MLBQSharedType.from_object([])


class MLBQConverterTest(parameterized.TestCase):
  _input_1d = {'it': 'was', 'the': 4.5, 'age': range(200)}
  _ans_1d = {'it': 'was', 'the': 4.5}
  for i in xrange(200):
    _ans_1d['age_{:03}'.format(i)] = i

  _input_2d = {'a': 'xyz', 'bee': 123}
  # shape == (11,3)
  _input_2d['q'] = np.arange(11)[:, np.newaxis] + 100 * np.arange(3)
  _ans_2d = {'a': 'xyz', 'bee': 123}
  for i in xrange(11):
    for j in xrange(3):
      _ans_2d['q_{:02}_{}'.format(i, j)] = 100 * j + i

  @parameterized.named_parameters(('1d', _input_1d, _ans_1d),
                                  ('2d', _input_2d, _ans_2d))
  def test_convert_to_bq(self, cloud_ml_line, answer):
    converter = MLBQConverter.from_object(cloud_ml_line)
    purported_answer = converter.convert_to_bigquery_dict(cloud_ml_line)
    self.assertEqual(purported_answer, answer)


if __name__ == '__main__':
  absltest.main()
