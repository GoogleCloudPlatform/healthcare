"""Helper to validate yaml configs against a schema."""

from absl import app
from absl import flags
from absl import logging

import jsonschema
import yaml

FLAGS = flags.FLAGS

flags.DEFINE_string('schema_path', None, 'Path to schema')
flags.DEFINE_string('config_path', None, 'Path to config')


def main(unused_argv):
  with open(FLAGS.schema_path, 'r') as f:
    schema = yaml.load(f, Loader=yaml.Loader)
  with open(FLAGS.config_path, 'r') as f:
    config = yaml.load(f, Loader=yaml.Loader)
  try:
    jsonschema.validate(config, schema)
  except jsonschema.exceptions.ValidationError:
    logging.info('Config: %s', config)
    raise


if __name__ == '__main__':
  app.run(main)
