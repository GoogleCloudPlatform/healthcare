"""Forseti provides utilities to manage Forseti instances."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import shutil
import tempfile

from utils import runner

_FORSETI_REPO = 'https://github.com/GoogleCloudPlatform/forseti-security.git'
_DEFAULT_BRANCH = 'dev'


def install(project_config):
  """Install a Forseti instance in the given project config.

  Args:
    project_config (ProjectConfig): project config of the project to deploy
        Forseti in.
  """

  tmp_dir = tempfile.mkdtemp()
  try:
    # clone repo
    runner.run_command(['git', 'clone', _FORSETI_REPO, tmp_dir])

    # make sure we're running from the default branch
    runner.run_command(['git', '-C', tmp_dir, 'checkout', _DEFAULT_BRANCH])

    # TODO: Pass in a project_id flag once
    # https://github.com/GoogleCloudPlatform/forseti-security/issues/2182
    # is closed.
    runner.run_command([
        'gcloud', 'config', 'set', 'project',
        project_config.project['project_id'],
    ])

    # run forseti installer
    runner.run_command([
        'python', os.path.join(tmp_dir, 'install/gcp_installer.py'),
        '--no-cloudshell',
    ], wait_for_output=False)
  finally:
    shutil.rmtree(tmp_dir)

