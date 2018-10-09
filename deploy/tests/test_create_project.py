"""Tests for healthcare.deploy.create_project.

These tests check that the module is free from syntax errors.
"""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import unittest

import create_project


class TestCreateProject(unittest.TestCase):

  def test_create_project(self):
    yaml_path = os.path.join(os.path.dirname(__file__),
                             '../samples/project_with_remote_audit_logs.yaml')
    parser = create_project.get_parser()
    create_project.main(parser.parse_args(['--project_yaml', yaml_path]))


if __name__ == '__main__':
  unittest.main()
