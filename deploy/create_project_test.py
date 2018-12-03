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

from deploy import create_project

FLAGS = flags.FLAGS


class CreateProjectTest(absltest.TestCase):

  def test_create_project_datathon(self):
    _deploy('datathon_team_project.yaml')

  def test_create_project_local_audit_logs(self):
    _deploy('project_with_local_audit_logs.yaml')

  def test_create_project_remote_audit_logs(self):
    _deploy('project_with_remote_audit_logs.yaml')


def _deploy(config_filename):
  FLAGS.project_yaml = os.path.join(
      FLAGS.test_srcdir, 'deploy/samples/',
      config_filename)
  with tempfile.NamedTemporaryFile() as f:
    FLAGS.output_yaml_path = f.name
    create_project.main([])


if __name__ == '__main__':
  absltest.main()
