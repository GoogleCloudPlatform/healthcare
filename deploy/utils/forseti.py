"""Forseti provides utilities to manage Forseti instances."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import shlex
import shutil
import tempfile

from utils import runner

_FORSETI_REPO = 'https://github.com/GoogleCloudPlatform/forseti-security.git'
_DEFAULT_BRANCH = 'dev'


def install(config):
  """Install a Forseti instance in the given project config.

  Args:
    config (dict): Forseti config dict of the Forseti instance to deploy.
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
        'gcloud', 'config', 'set', 'project', config['project']['project_id'],
    ])

    # run forseti installer
    install_cmd = [
        'python', os.path.join(tmp_dir, 'install/gcp_installer.py'),
        '--no-cloudshell',
    ]
    if 'installer_flags' in config:
      install_cmd.extend(shlex.split(config['installer_flags']))

    runner.run_command(install_cmd, wait_for_output=False)
  finally:
    shutil.rmtree(tmp_dir)
