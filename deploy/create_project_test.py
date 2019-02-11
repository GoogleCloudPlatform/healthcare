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
from backports import tempfile

import ruamel.yaml

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

  def test_project_config_validate_check_raise(self):
    FLAGS.projects = ['*']
    datathon_path = (
        'deploy/samples/datathon_team_project.yaml'
    )
    root_config = utils.read_yaml_file(datathon_path)
    utils.resolve_env_vars(root_config)
    root_config['overall']['allowed_apis'] = []
    with tempfile.TemporaryDirectory() as tmp_dir:
      FLAGS.output_yaml_path = os.path.join(tmp_dir, 'conf.yaml')
      FLAGS.project_yaml = FLAGS.output_yaml_path
      FLAGS.output_cleanup_path = os.path.join(tmp_dir, 'cleanup.sh')
      with open(FLAGS.project_yaml, 'w') as f:
        yaml = ruamel.yaml.YAML()
        yaml.dump(root_config, f)
        f.flush()
      with self.assertRaises(utils.InvalidConfigError):
        create_project.main([])

  def test_project_config_validate_check_correct(self):
    FLAGS.projects = ['*']
    datathon_path = (
        'deploy/samples/datathon_team_project.yaml'
    )
    root_config = utils.read_yaml_file(datathon_path)
    utils.resolve_env_vars(root_config)
    root_config['overall']['allowed_apis'] = [
        'bigquery-json.googleapis.com',
        'compute.googleapis.com',
        'ml.googleapis.com',
    ]

    with tempfile.TemporaryDirectory() as tmp_dir:
      FLAGS.output_yaml_path = os.path.join(tmp_dir, 'conf.yaml')
      FLAGS.project_yaml = FLAGS.output_yaml_path
      FLAGS.output_cleanup_path = os.path.join(tmp_dir, 'cleanup.sh')
      with open(FLAGS.project_yaml, 'w') as f:
        yaml = ruamel.yaml.YAML()
        yaml.dump(root_config, f)
        f.flush()
      create_project.main([])

  def test_create_project_with_spanned_configs(self):
    FLAGS.project_yaml = (
        'deploy/samples/spanned_configs/root.yaml')
    FLAGS.projects = ['*']
    with tempfile.TemporaryDirectory() as tmp_dir:
      FLAGS.output_yaml_path = os.path.join(tmp_dir, 'out.yaml')
      FLAGS.output_cleanup_path = os.path.join(tmp_dir, 'cleanup.sh')
      create_project.main([])


def _deploy(config_filename):
  FLAGS.project_yaml = os.path.join(
      'deploy/samples/', config_filename)
  FLAGS.projects = ['*']
  with tempfile.TemporaryDirectory() as tmp_dir:
    FLAGS.output_yaml_path = os.path.join(tmp_dir, 'out.yaml')
    FLAGS.output_cleanup_path = os.path.join(tmp_dir, 'cleanup.sh')
    create_project.main([])


if __name__ == '__main__':
  create_project._IAM_PROPAGATAION_WAIT_TIME_SECS = 0  # don't sleep
  absltest.main()
