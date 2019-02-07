"""Load feature vectors from Cloud ML Engine batch prediction into BigQuery.

Cloud ML Engine's batch predict mode outputs a text file with a JSON instance on
each line that encodes the output of the model for each input.
This script creates a BigQuery table from this output.
"""
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import json

import apache_beam as beam
from apache_beam.io.gcp.bigquery import BigQueryDisposition
from apache_beam.options.pipeline_options import PipelineOptions
from apache_beam.options.pipeline_options import SetupOptions
import numpy as np
import tensorflow as tf


class Parse(beam.DoFn):
  """Parses the text output of Cloud ML Engine (CMLE) batch prediction."""

  def __init__(self, converter):
    self.converter = converter

  def process(self, element):
    """Overrides beam.DoFn.process.

    Args:
      element (str): one line of Cloud ML Engine batch prediction output

    Yields:
      Dict[str, Any]: a dictionary that maps column names to their value.
    """
    yield self.converter.convert_to_bigquery_dict(json.loads(element))


def run(text_input, bq_dataset, bq_table, pipeline_options):
  """Build and execute the Apache Beam pipeline.

  Args:
    text_input (str): A text file output by Cloud ML Engine batch prediction.
      E.g. using the model defined in build_cloudml_model.ipynb
    bq_dataset (str): The name of the BigQuery dataset to create the table in.
    bq_table (str): The name of the table to create. If the table already
      exists, it will be overwritten.
    pipeline_options (PipelineOptions): Apache Beam specific command line
      arguments. see https://beam.apache.org/documentation/runners/dataflow/
  """
  with tf.gfile.GFile(text_input) as fp:
    line_0 = fp.readline()
  ml_to_bq_converter = MLBQConverter.from_object(json.loads(line_0))
  bq_schema = ml_to_bq_converter.to_bigquery_schema()
  with beam.Pipeline(options=pipeline_options) as p:
    _ = (
        p
        | beam.io.ReadFromText(text_input)
        | beam.ParDo(Parse(ml_to_bq_converter))
        | beam.io.gcp.bigquery.WriteToBigQuery(
            table=bq_table,
            dataset=bq_dataset,
            schema=bq_schema,
            write_disposition=BigQueryDisposition.WRITE_TRUNCATE))


class MLBQSharedType(object):
  """A Cloud ML engine and BigQuery type.

  Represents the type of an attribute output by Cloud ML Engine batch
  predicition, and its corresponding type in BigQuery. Supported types are
  integers, strings, and doubles, and tensors thereof.

  Args:
    py_type (type): The corresponding python type that mediates the conversion.
      Can be int, float, str, or unicode.
    shape (tuple): the shape of the tensor. The empty tuple stands for scalars.
  """
  supported_types = {
      int: 'INT64',
      float: 'FLOAT64',
      str: 'STRING',
      unicode: 'STRING'
  }

  def __init__(self, py_type, shape):
    self.py_type = py_type
    self.shape = shape
    self.idx_fmt_string = '_'.join('{:0%d}' % len(str(d)) for d in shape)

  @classmethod
  def from_object(cls, obj):
    """Create a MLBQSharedType by inferring the type of a python object.

    Args:
      obj (Union[int, float, str, unicode, list]): an attribute value of Cloud
        ML output. This can be created by parsing the json entity on a line of
        Cloud ML text output and selecting one of the attributes.

    Raises:
      ValueError: when passed an empty tensor.
      NotImplementedError: when passed a tensor of unsupported type. See
      MLBQSharedType.supported_types.keys() for a list of supported types.
    Returns:
      MLBQSharedType: represents the type of the passed object.
    """
    arr = np.array(obj)
    if arr.size == 0:
      raise ValueError('Cannot create an MLBQSharedType from an empty iterable')
    example_value = arr.flat[0].item()
    py_type = type(example_value)
    if py_type not in cls.supported_types:
      raise NotImplementedError('Unsupported type: {} for object {}'.format(
          py_type, example_value))
    shape = arr.shape
    return cls(py_type, shape)

  def to_bigquery_schema(self, name):
    """Get the represenation of this type as columns in a BigQuery schema.

    Args:
      name (str): the name of the column(s).

    Returns:
      str: A string representing a BigQuery schema as described in
      https://cloud.google.com/bigquery/docs/quickstarts/quickstart-command-line
      Tensors columns are named with the given name as their prefix, followed by
      an underscore, followed by their indices (starting from zero) each
      separated by an underscore.
    """
    bq_type = self.supported_types[self.py_type]
    if not self.shape:
      # Scalar type
      return '{}:{}'.format(name, bq_type)
    else:
      columns = []
      for idx in np.ndindex(self.shape):
        idx_str = self.idx_fmt_string.format(*idx)
        columns.append('{}_{}:{}'.format(name, idx_str, bq_type))
      return ','.join(columns)


class MLBQConverter(object):
  """Converts Cloud ML batch prediction output to dictionaries for BigQuery.

  Args:
    attributes (Dict[str, MLBQSharedType]): A dictionary mapping column names to
      their MLBQSharedType.
  """

  def __init__(self, attributes):
    self._data = attributes

  @classmethod
  def from_object(cls, obj):
    """Create a MLBQConverter from a parsed line of Cloud ML output.

    Args:
      obj (Dict[str, Union[int, float, string, list]]): A single parsed line of
        JSON Cloud ML batch prediction output. Types are sizes are inferred from
        this line.

    Returns:
      MLBQConverter: Converts Cloud ML output to the inferred BigQuery schema.
    """
    attributes = {
        name: MLBQSharedType.from_object(value)
        for name, value in obj.iteritems()
    }
    return cls(attributes)

  def to_bigquery_schema(self):
    """Get the schema of the BigQuery table this object converts to.

    Returns:
      str: A string representation of the BigQuery schema. The details of the
      syntax for this string can be found here
      https://cloud.google.com/bigquery/docs/quickstarts/quickstart-command-line
    """
    return ','.join(
        sorted(
            shared_type.to_bigquery_schema(name)
            for name, shared_type in self._data.iteritems()))

  def convert_to_bigquery_dict(self, cloud_ml_dict):
    """Convert a Cloud ML output dictionary into a BigQuery input dictionary.

    Args:
      cloud_ml_dict: A dictionary obtained by parsing a single line of JSON
        Cloud ML batch prediction output.

    Returns:
      Dict[str, Union[int, float, str, unicode]]: A dictionary suitable for
      writing to the inferred BigQuery schema with
      apache_beam.io.gcp.bigquery.WriteToBigQuery
    """
    bq_dict = dict()
    for name, value in cloud_ml_dict.iteritems():
      conv_type = self._data[name]
      if not conv_type.shape:
        # Scalar type: no expansion needed
        bq_dict[name] = value
      else:
        iterator = np.nditer(np.array(value), ['multi_index'])
        for elem in iterator:
          idx_str = conv_type.idx_fmt_string.format(*iterator.multi_index)
          bq_dict['{}_{}'.format(name, idx_str)] = elem.item()
    return bq_dict


if __name__ == '__main__':
  parser = argparse.ArgumentParser(description=__doc__)
  parser.add_argument(
      '--input_text_file',
      required=True,
      help="""A text file outputted by cloud ML Engine. See
      build_cloudml_model.ipynb for and example of how to generate this file.
      This file should contain a JSON instance on each line, conaining
      attributes that are integers, doubles, strings or tensors thereof.""")

  parser.add_argument(
      '--bq_dataset',
      required=True,
      help='The name of the BigQuery dataset create the table in.')

  parser.add_argument(
      '--bq_table',
      required=True,
      help="""The name of the table to create. If this table already exists it
      will be overwritten.""")

  args, pipeline_args = parser.parse_known_args()
  beam_options = PipelineOptions(pipeline_args)
  # serialize and provide global imports, functions, etc. to workers.
  beam_options.view_as(SetupOptions).save_main_session = True

  if (beam_options.runner == 'DataflowRunner' and
      not args.input_text_file.startswith('gs://')):
    parser.error('--input_text_file must start with gs:// when using '
                 'DataflowRunner')
  run(args.input_text_file, args.bq_dataset, args.bq_table, beam_options)
