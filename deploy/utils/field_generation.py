# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""field_generation provides utilities to manage generated_fields."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

from deploy.utils import utils

# The tag name of generated_fields.
GENERATED_FIELDS_NAME = 'generated_fields'


def update_generated_fields(input_yaml_path, new_config):
  """Get and update the generated_fields block of the input yaml."""
  cfg_content = utils.read_yaml_file(input_yaml_path)
  if GENERATED_FIELDS_NAME not in new_config:
    cfg_content.pop(GENERATED_FIELDS_NAME, {})
  else:
    cfg_content[GENERATED_FIELDS_NAME] = new_config[GENERATED_FIELDS_NAME]
  return cfg_content


def rewrite_generated_fields_back(project_yaml, new_config):
  """Write config file to output_yaml_path with new generated_fields."""
  cfg_content = update_generated_fields(project_yaml, new_config)
  utils.write_yaml_file(cfg_content, project_yaml)
