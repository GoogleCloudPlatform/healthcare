# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

r"""Utility to generate Forseti scanner rules given project configurations.

Usage:
  bazel run :generate_rules -- \
      --deployment_config_path="${DEPLOYMENT_CONFIG_PATH}" \
      --output_path="${OUTPUT_PATH}"
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl import app
from absl import flags

from deploy.rule_generator import rule_generator
from deploy.utils import utils

FLAGS = flags.FLAGS

flags.DEFINE_string('deployment_config_path', None,
                    'YAML file containing all project configurations.')
flags.DEFINE_string('output_path', None,
                    ('Path to local directory or GCS bucket to output rules '
                     ' files. If unset, directly writes to the Forseti server '
                     'bucket.'))


def main(argv):
  del argv  # Unused.
  deployment_config = utils.read_yaml_file(
      utils.normalize_path(FLAGS.deployment_config_path))
  output_path = (utils.normalize_path(FLAGS.output_path)
                 if FLAGS.output_path else None)
  rule_generator.run(deployment_config, output_path=output_path)

if __name__ == '__main__':
  flags.mark_flag_as_required('deployment_config_path')
  app.run(main)
