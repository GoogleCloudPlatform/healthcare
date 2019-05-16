"""An Apache Beam DoFn for lazily collecting GCS paths into a PCollection."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import re
import apache_beam as beam
from google.cloud import storage

_SPLIT_INTO_BUCKET_AND_BLOB_REGEX = re.compile(r'^gs://([a-z0-9\-_.]+)(|/.*)$')


def split_into_bucket_and_blob(path):
  """Split a GCS URI into a bucket name and a blob name.

  Args:
    path: A GCS path starting with gs:// and the full name of a bucket.

  Returns:
    Tuple[str, str]: The name of the bucket and the blob within the bucket.
  """
  path_match = _SPLIT_INTO_BUCKET_AND_BLOB_REGEX.match(path)
  if path_match:
    bucket_name, blob_name = path_match.groups()
    if blob_name.startswith('/'):
      blob_name = blob_name[1:]
  else:
    raise ValueError('Invalid GCS path or path prefix' + path)
  return bucket_name, blob_name


class GetPaths(beam.DoFn):
  """Collect all the paths in a GCS bucket with a given prefix.

  Notes:
    Since this operation is a "fan-out" type operation, it should generally
    be followed by a beam.ReShuffle transform to avoid spurious fusion.

  Args:
    max_results (Union[int, None], default: None): the maximum number of paths
      to yield per prefix
    validation_regex (Union[str, None], default: None): a regular expression
      that paths, excluding the gs://{bucket}/ prefix, must match in order to be
        added to the output PCollection
  """

  def __init__(self, max_results=None, validation_regex=None):
    super(GetPaths, self).__init__()
    self.max_results = max_results
    # Not serializable, to be initialized on each worker:
    self._client = None
    self.validation_regex = re.compile(
        validation_regex) if validation_regex else None

  def process(self, prefix):
    """Overrides beam.DoFn.process.

    Args:
      prefix (str): the prefix that generated paths share. Begins with gs:// and
        a the complete name of a GCS bucket.

    Yields:
      unicode: all GCS blob URIs that start with the string
      gs://{self.bucket}/{prefix} and match validation_regex if it was specified
    """
    if self._client is None:
      self._client = storage.Client()

    bucket_name, blob_prefix = split_into_bucket_and_blob(prefix)
    for blob in self._client.bucket(bucket_name).list_blobs(
        prefix=blob_prefix, max_results=self.max_results):
      if not blob.name.endswith('/'):
        uri = u'gs://{}/{}'.format(bucket_name, blob.name)
        if self.validation_regex is None or self.validation_regex.search(uri):
          yield uri
