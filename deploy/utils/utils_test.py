"""Tests for healthcare.deploy.utils.utils.py.

These tests check that the module is free from syntax errors.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os

from absl import flags
from absl.testing import absltest

from deploy.utils import utils

FLAGS = flags.FLAGS


class UtilsTest(absltest.TestCase):

  def test_load_config_spanned_configs(self):
    project_yaml = os.path.join(
        FLAGS.test_srcdir,
        'deploy/samples/',
        'project_with_remote_audit_logs.yaml')
    input_yaml_path = utils.normalize_path(project_yaml)
    dict1 = utils.load_config(input_yaml_path)

    project_yaml = os.path.join(
        FLAGS.test_srcdir,
        'deploy/samples/spanned_configs',
        'root.yaml')
    input_yaml_path = utils.normalize_path(project_yaml)
    dict2 = utils.load_config(input_yaml_path)

    self.assertEqual(dict1, dict2)


if __name__ == '__main__':
  absltest.main()
