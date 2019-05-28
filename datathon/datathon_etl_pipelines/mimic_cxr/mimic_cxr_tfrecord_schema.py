"""Defines the format of the TFRecords produces by the MIMIC CXR ingestion pipeline."""
import itertools
from datathon_etl_pipelines.mimic_cxr.enum_encodings import ID_NAMES
from datathon_etl_pipelines.mimic_cxr.enum_encodings import LABEL_NAMES
from datathon_etl_pipelines.utils import bytes_feature
from datathon_etl_pipelines.utils import int64_feature
import tensorflow as tf

FEATURE_DESCRIPTION = {'jpg_bytes': tf.io.FixedLenFeature([], tf.string)}
for feature_name in itertools.chain(LABEL_NAMES, ID_NAMES):
  FEATURE_DESCRIPTION[feature_name] = tf.io.FixedLenFeature([], tf.int64)


def build_features(jpg_bytes, ids, labels):
  """Create a dictionary of features for building a MIMIC CXR TFRecord.

  Args:
    jpg_bytes (bytes): The contents of the jpg image for this instance.
    ids (Tuple[int, int, int, int, int]): A tuple of integers identifiers of the
      form  (patient, study, image, view, dataset). The dataset value is not
      used and may be omitted.
    labels (List[int]): the values of the labels in the same order as
      `datathon_etl_pipelines.mimic_cxr.enum_encodings.LABEL_NAMES`

  Returns:
    Dict[str, tf.train.Feature]:  A dictionary of features that can be used to
    construct a TFExample.
  """
  features = {'jpg_bytes': bytes_feature(jpg_bytes)}

  for name, value in zip(ID_NAMES, ids):
    features[name] = int64_feature(value)
  for label, value in zip(LABEL_NAMES, labels):
    features[label] = int64_feature(value)

  return features
