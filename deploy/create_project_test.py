"""Tests for healthcare.deploy.create_project.

These tests check that the module is free from syntax errors.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os

from absl import flags
from absl.testing import absltest

from deploy import create_project

FLAGS = flags.FLAGS


class TestCreateProject(absltest.TestCase):

  def test_create_project(self):
    FLAGS.project_yaml = os.path.join(
        os.path.dirname(__file__),
        '../samples/project_with_remote_audit_logs.yaml')
    create_project.main([])

if __name__ == '__main__':
  absltest.main()
