import itertools

from absl.testing import absltest
from absl.testing import parameterized
from datathon_etl_pipelines.generic_imaging.images_to_tfrecord import ConvertToTFExample

# [(test_case_name, image_format, input_join, expected_list)]
tf_example_conversion_cases = [('one_to_one_jpg', 'jpg', (u'path_value', {
    'images': [b'bytestring'],
    'labels': [{
        u'a': 1,
        u'b': 3.0,
        u'c': u'd'
    }]
}), [{
    u'path': b'path_value',
    u'jpg_bytes': b'bytestring',
    u'a': 1,
    u'b': 3.0,
    u'c': b'd'
}]),
                               ('one_to_one_png', 'png', (u'path_value', {
                                   'images': [b'bytestring'],
                                   'labels': [{
                                       u'a': 1,
                                       u'b': 3.0,
                                       u'c': u'd'
                                   }]
                               }), [{
                                   u'path': b'path_value',
                                   u'png_bytes': b'bytestring',
                                   u'a': 1,
                                   u'b': 3.0,
                                   u'c': b'd'
                               }]),
                               ('inner_join_property', 'png', (u'path_value', {
                                   'images': [b'bytestring'],
                                   'labels': []
                               }), []),
                               ('cartesian_product_property', 'jpg',
                                (u'path_value', {
                                    'images': [b'bytestring'],
                                    'labels': [{
                                        u'a': 1,
                                        u'b': 3.0,
                                        u'c': u'd'
                                    }, {
                                        u'x': -3,
                                        u'y': 4
                                    }]
                                }), [{
                                    u'path': b'path_value',
                                    u'jpg_bytes': b'bytestring',
                                    u'a': 1,
                                    u'b': 3.0,
                                    u'c': b'd'
                                }, {
                                    u'path': b'path_value',
                                    u'jpg_bytes': b'bytestring',
                                    u'x': -3,
                                    u'y': 4
                                }])]


class ConvertToTFExampleTest(parameterized.TestCase):

  def tfexample_to_dict(self, tfexample):
    result = dict()
    for name, list_proto in tfexample.features.feature.items():
      values = list(
          itertools.chain(list_proto.int64_list.value,
                          list_proto.float_list.value,
                          list_proto.bytes_list.value))
      self.assertLen(values, 1)
      result[name] = values[0]
    return result

  @parameterized.named_parameters(*tf_example_conversion_cases)
  def test_conversion(self, image_format, input_join, expected_list):
    dofn = ConvertToTFExample(image_format)
    result_list = [
        self.tfexample_to_dict(result) for result in dofn.process(input_join)
    ]
    self.assertSameElements(result_list, expected_list)


if __name__ == '__main__':
  absltest.main()
