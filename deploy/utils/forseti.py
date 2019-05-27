"""Forseti provides utilities to manage Forseti instances."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import collections
import os
import re
import shlex
import shutil
import tempfile

from absl import flags

from deploy.utils import runner

FLAGS = flags.FLAGS

_FORSETI_REPO = 'https://github.com/GoogleCloudPlatform/forseti-security.git'
_DEFAULT_BRANCH = 'dev'
_FORSETI_SERVER_SERVICE_ACCOUNT_FILTER = 'email:forseti-server-gcp-*'
_FORSETI_SERVER_BUCKET_RE = re.compile(r'gs://forseti-server-.*')


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
    # TODO: switch to py3 call when Forseti supports it
    # (https://github.com/GoogleCloudPlatform/forseti-security/issues/1437)
    # or use the DM template once
    # https://github.com/GoogleCloudPlatform/forseti-security/issues/2544
    # is fixed.
    install_cmd = [
        'python2',
        os.path.join(tmp_dir, 'install/gcp_installer.py'),
        '--no-cloudshell',
    ]
    if 'installer_flags' in config:
      install_cmd.extend(shlex.split(config['installer_flags']))

    # Run installer in the temp dir in case cwd is readonly.
    cwd = os.getcwd()
    os.chdir(tmp_dir)
    runner.run_command(install_cmd)
    os.chdir(cwd)
  finally:
    shutil.rmtree(tmp_dir)


def get_server_service_account(forseti_project_id):
  """Get the service account for the Forseti server instance.

  Assumes there is only one Forseti instance installed in the project.

  Args:
    forseti_project_id (str): id of the Forseti project.

  Returns:
    str: the forseti server service account.

  Raises:
    ValueError: if gcloud returns an unexpected number of service accounts.
  """
  output = runner.run_gcloud_command([
      'iam', 'service-accounts', 'list',
      '--format', 'value(email)',
      '--filter', _FORSETI_SERVER_SERVICE_ACCOUNT_FILTER,
  ], project_id=forseti_project_id)

  service_accounts = output.strip().split('\n')
  if len(service_accounts) != 1:
    raise ValueError(
        ('Unexpected number of Forseti server service accounts: '
         'got {}, want 1, {}'.format(len(service_accounts), output)))
  return service_accounts[0]


def get_server_bucket(forseti_project_id):
  """Get the bucket holding the Forseti server instance's configuration.

  Args:
    forseti_project_id (str): id of the Forseti project.

  Returns:
    str: the forseti server bucket name.

  Raises:
    ValueError: if failure in finding the bucket in the gsutil command output.
  """
  output = runner.run_command(['gsutil', 'ls', '-p', forseti_project_id],
                              get_output=True)

  match = _FORSETI_SERVER_BUCKET_RE.search(output)
  if not match:
    raise ValueError('Failed to find Forseti server bucket: {}'.format(output))
  return match.group(0)


# Standard (built in) roles required by the Forseti service account on
# projects to be monitored.
_STANDARD_ROLES = [
    'appengine.appViewer',
    'browser',
    'compute.networkViewer',
    'iam.securityReviewer',
    'servicemanagement.quotaViewer',
    'serviceusage.serviceUsageConsumer',
]

CustomRole = collections.namedtuple(
    'CustomRole', ['name', 'title', 'description', 'permissions'])

# Custom roles required by the Forseti service account on projects to be
# monitored.
_CUSTOM_ROLES = [
    CustomRole(
        name='forsetiBigqueryViewer',
        title='Forseti BigQuery Metadata Viewer',
        description='Access to only view BigQuery datasets and tables',
        permissions=[
            'bigquery.datasets.get',
            'bigquery.tables.get',
            'bigquery.tables.list',
        ],
    ),
    CustomRole(
        name='forsetiCloudsqlViewer',
        title='Forseti CloudSql Metadata Viewer',
        description='Access to only view CloudSql resources',
        permissions=[
            'cloudsql.backupRuns.get',
            'cloudsql.backupRuns.list',
            'cloudsql.databases.get',
            'cloudsql.databases.list',
            'cloudsql.instances.get',
            'cloudsql.instances.list',
            'cloudsql.sslCerts.get',
            'cloudsql.sslCerts.list',
            'cloudsql.users.list',
        ],
    ),
]


def get_forseti_roles(project_id):
  """Gets a list of all roles granted to the Forseti service account."""
  standard_roles = ['roles/{}'.format(role) for role in _STANDARD_ROLES]
  custom_roles = [
      'projects/{}/roles/{}'.format(project_id, role.name)
      for role in _CUSTOM_ROLES
  ]
  return standard_roles + custom_roles


def grant_access(project_id, forseti_service_account):
  """Grant the necessary permissions to the Forseti service account."""
  for role in _STANDARD_ROLES:
    _add_binding(project_id, forseti_service_account, 'roles/{}'.format(role))

  for custom_role in _CUSTOM_ROLES:
    _create_custom_role(custom_role, project_id)
    _add_binding(project_id, forseti_service_account,
                 'projects/{}/roles/{}'.format(project_id, custom_role.name))


def _add_binding(project_id, forseti_service_account, role):
  """Add an IAM Policy for the Forseti service account for the given role."""
  cmd = [
      'projects', 'add-iam-policy-binding',
      project_id,
      '--member', 'serviceAccount:{}'.format(forseti_service_account),
      '--role', role,
  ]
  runner.run_gcloud_command(cmd, project_id=None)


def _create_custom_role(custom_role, project_id):
  """Create a custom IAM role in the project."""
  cmd = [
      'iam', 'roles', 'create', custom_role.name,
      '--project', project_id,
      '--title', custom_role.title,
      '--description', custom_role.description,
      '--stage', 'ALPHA',
      '--permissions', ','.join(custom_role.permissions),
  ]
  runner.run_gcloud_command(cmd, project_id=None)
