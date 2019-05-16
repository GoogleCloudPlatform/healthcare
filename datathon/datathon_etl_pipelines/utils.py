"""Utility functions that are used by the datathon ETL pipelines."""
import inspect
import os
import tensorflow as tf


def get_setup_file():
  """Returns the absolute path to the setup file for this package.

  Apache Beam requires this setup file to build this package on remote workers.
  """
  this_file = inspect.getfile(inspect.currentframe())
  this_dir = os.path.dirname(this_file)
  relative_path = os.path.join(this_dir, '..', 'setup.py')
  return os.path.abspath(relative_path)


def get_test_data():
  """Returns the absolute path to the setup file for this package.

  Apache Beam requires this setup file to build this package on remote workers.
  """
  this_file = inspect.getfile(inspect.currentframe())
  this_dir = os.path.dirname(this_file)
  relative_path = os.path.join(this_dir, 'test_data')
  return os.path.abspath(relative_path)


def bytes_feature(value):
  """Returns a bytes_list from a string / byte."""
  return tf.train.Feature(bytes_list=tf.train.BytesList(value=[value]))


def float_feature(value):
  """Returns a float_list from a float / double / decimal."""
  return tf.train.Feature(float_list=tf.train.FloatList(value=[value]))


def int64_feature(value):
  """Returns an int64_list from a bool / int / uint."""
  return tf.train.Feature(int64_list=tf.train.Int64List(value=[value]))


def scalar_to_feature(value):
  """Converts a polymorphic value to a a tf.train.Feature.


  bool, int, uint -> int64
  float, double, decimal -> float
  string, bytes -> bytes

  This supports every type returned by Apache Beam's BigQuery Data Source,
  except for records and arrays.

  Args:
    value (Union[object, np.array, tf.Tensor]): the value to be converted.

  Returns:
    tf.train.Feature: the converted value.
  """
  try:
    return int64_feature(value)
  except TypeError:
    try:
      return float_feature(value)
    except TypeError:
      if isinstance(value, unicode):
        return bytes_feature(value.encode())
      else:
        return bytes_feature(value)
