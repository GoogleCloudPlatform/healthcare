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

import yaml

from deploy import create_project
from deploy.utils import utils

FLAGS = flags.FLAGS


class CreateProjectTest(absltest.TestCase):

  def test_create_project_datathon(self):
    _deploy('datathon_team_project.yaml')

  def test_create_project_local_audit_logs(self):
    _deploy('project_with_local_audit_logs.yaml')

  def test_create_project_remote_audit_logs(self):
    _deploy('project_with_remote_audit_logs.yaml')

  def test_project_config_validate_check_correct(self):
    datathon_path = os.path.join(
        FLAGS.test_srcdir,
        'google3/deploy/samples/',
        'datathon_team_project.yaml')
    root_config = utils.resolve_env_vars(utils.read_yaml_file(datathon_path))
    root_config['overall']['allowed_apis'] = [
        'bigquery-json.googleapis.com',
        'compute.googleapis.com',
        'ml.googleapis.com',
    ]
    root_config_str = yaml.dump(root_config)
    with tempfile.NamedTemporaryFile(suffix='.yaml', mode='w') as f:
      f.write(root_config_str)
      f.flush()
      FLAGS.project_yaml = f.name
      with tempfile.NamedTemporaryFile() as fout:
        FLAGS.output_yaml_path = fout.name
        create_project.main([])

  def test_create_project_with_spanned_configs(self):
    FLAGS.project_yaml = os.path.join(
        FLAGS.test_srcdir,
        'google3/deploy/samples/spanned_configs',
        'root.yaml')
    with tempfile.NamedTemporaryFile() as f:
      FLAGS.output_yaml_path = f.name
      create_project.main([])


def _deploy(config_filename):
  FLAGS.project_yaml = os.path.join(
      FLAGS.test_srcdir, 'google3/deploy/samples/',
      config_filename)
  with tempfile.NamedTemporaryFile() as f:
    FLAGS.output_yaml_path = f.name
    create_project.main([])


if __name__ == '__main__':
  absltest.main()
