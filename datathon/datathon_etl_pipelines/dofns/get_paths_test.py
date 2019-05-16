from absl.testing import absltest
from absl.testing import parameterized
from datathon_etl_pipelines.dofns.get_paths import split_into_bucket_and_blob


class SplitIntoBucketAndBlobTest(parameterized.TestCase):

  @parameterized.named_parameters(
      ('just_bucket', 'gs://bucket-name', 'bucket-name', ''),
      ('bucket_obj', 'gs://bucket/test_obj.txt', 'bucket', 'test_obj.txt'),
      ('bucket_path', 'gs://b/this/that/those.csv', 'b', 'this_that_those.txt'),
  )
  def test_over_valid_inputs(self, uri, true_bucket_name, true_blob_name):
    bucket_name, blob_name = split_into_bucket_and_blob(uri)
    self.assertEqual(bucket_name, true_bucket_name)
    self.assertEqual(blob_name, true_blob_name)

  @parameterized.named_parameters(
      ('empty', 'gs://'), ('no_bucket', 'gs:///test_obj.txt'),
      ('capital_bucket', 'gs://B/this/that/those.csv'),
      ('no_gs', '/this/local/path.txt'))
  def test_over_invalid_inputs(self, uri):
    with self.assertRaises(ValueError):
      _ = split_into_bucket_and_blob(uri)


if __name__ == '__main__':
  absltest.main()
