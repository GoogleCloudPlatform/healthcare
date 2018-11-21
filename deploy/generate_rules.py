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

"""Utility to generate Forseti scanner rules given project configurations.

Usage:
  rule_generator
      --project_configs="${PROJECT_CONFIGS}"
      --forseti_rules_dir="${FORSETI_RULES_DIR}"
      --forseti_gcp_reader="${FORSETI_GCP_READER}"
      --alsologtostderr
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from absl import app
from absl import flags

from deploy.rule_generator import rule_generator

FLAGS = flags.FLAGS

flags.DEFINE_string('project_config', None,
                    'YAML file containing all project configurations.')
flags.DEFINE_string('forseti_rules_dir', None, 'Path of output rules files.')


def main(argv):
  del argv  # Unused.
  rule_generator.run(FLAGS.project_config, FLAGS.forseti_rules_dir)

if __name__ == '__main__':
  flags.mark_flag_as_required('project_config')
  flags.mark_flag_as_required('forseti_rules_dir')
  app.run(main)
