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
"""Tests for healthcare.deploy.create_project.

These tests check that the module is free from syntax errors.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import tempfile

from absl import flags
from absl.testing import absltest

import ruamel.yaml

from deploy import create_project
from deploy.utils import utils

FLAGS = flags.FLAGS


class CreateProjectTest(absltest.TestCase):

  def test_create_project_local_audit_logs(self):
    _deploy('project_with_local_audit_logs.yaml')

  def test_create_project_remote_audit_logs(self):
    _deploy('project_with_remote_audit_logs.yaml')

  def test_create_project_with_spanned_configs(self):
    _deploy('spanned_configs/root.yaml')

  def test_project_config_validate_check_raise(self):
    FLAGS.projects = ['*']
    path = (
        'deploy/samples/project_with_local_audit_logs.yaml'
    )
    root_config = utils.read_yaml_file(path)
    root_config['overall']['allowed_apis'] = []
    root_config['projects'][0]['enabled_apis'] = ['foo.googleapis.com']
    with tempfile.TemporaryDirectory() as tmp_dir:
      FLAGS.project_yaml = os.path.join(tmp_dir, 'conf.yaml')
      FLAGS.generated_fields_path = os.path.join(tmp_dir, 'generated.yaml')
      with open(FLAGS.project_yaml, 'w') as f:
        yaml = ruamel.yaml.YAML()
        yaml.dump(root_config, f)
        f.flush()
      with self.assertRaises(utils.InvalidConfigError):
        create_project.main([])

  def test_project_config_validate_check_correct(self):
    FLAGS.projects = ['*']
    path = (
        'deploy/samples/project_with_local_audit_logs.yaml'
    )
    root_config = utils.read_yaml_file(path)
    root_config['overall']['allowed_apis'] = [
        'bigquery-json.googleapis.com',
        'compute.googleapis.com',
        'ml.googleapis.com',
    ]

    with tempfile.TemporaryDirectory() as tmp_dir:
      FLAGS.project_yaml = os.path.join(tmp_dir, 'conf.yaml')
      FLAGS.generated_fields_path = os.path.join(tmp_dir, 'generated.yaml')
      with open(FLAGS.project_yaml, 'w') as f:
        yaml = ruamel.yaml.YAML()
        yaml.dump(root_config, f)
        f.flush()
      create_project.main([])

  def test_get_data_bucket_name(self):
    data_bucket = {
        'name': 'my-project-data1',
        'storage_class': 'MULTI_REGIONAL',
        'location': 'US'
    }
    bucket_name = create_project.get_data_bucket_name(data_bucket, 'my-project')
    self.assertEqual(bucket_name, 'my-project-data1')

    data_bucket = {
        'name_suffix': '-data2',
        'storage_class': 'MULTI_REGIONAL',
        'location': 'US'
    }
    bucket_name = create_project.get_data_bucket_name(data_bucket, 'my-project')
    self.assertEqual(bucket_name, 'my-project-data2')

    data_bucket = {'storage_class': 'MULTI_REGIONAL', 'location': 'US'}
    with self.assertRaises(utils.InvalidConfigError):
      create_project.get_data_bucket_name(data_bucket, 'my-project')

    data_bucket = {
        'name': 'my-project-data3',
        'name_suffix': '-data2',
        'storage_class': 'MULTI_REGIONAL',
        'location': 'US'
    }
    with self.assertRaises(utils.InvalidConfigError):
      create_project.get_data_bucket_name(data_bucket, 'my-project')


def _deploy(config_filename):
  FLAGS.project_yaml = os.path.join(
      'deploy/samples/', config_filename)
  FLAGS.projects = ['*']
  with tempfile.TemporaryDirectory() as tmp_dir:
    FLAGS.generated_fields_path = os.path.join(tmp_dir, 'generated.yaml')
    create_project.main([])


if __name__ == '__main__':
  absltest.main()
