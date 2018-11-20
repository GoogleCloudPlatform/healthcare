"""Tests for healthcare.deploy.utils.forseti."""

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import subprocess

from absl import flags
from absl.testing import absltest

import mock

from deploy.utils import forseti

FLAGS = flags.FLAGS


class ForsetiAccessTest(absltest.TestCase):

  @mock.patch.object(subprocess, 'check_output', return_value=b'foo')
  def test_grant_access(self, mock_check_output):
    FLAGS.dry_run = False
    forseti.grant_access(
        'project1', 'forseti-sa@@forseti-project.iam.gserviceaccount.com')

    want_calls = []

    want_calls.extend(
        build_add_binding_call('roles/{}'.format(role))
        for role in forseti._STANDARD_ROLES
    )

    want_calls.append(mock.call([
        'gcloud', 'iam', 'roles', 'create', 'forsetiBigqueryViewer',
        '--project', 'project1',
        '--title', 'Forseti BigQuery Metadata Viewer',
        '--description', 'Access to only view BigQuery datasets and tables',
        '--stage', 'ALPHA',
        '--permissions',
        'bigquery.datasets.get,bigquery.tables.get,bigquery.tables.list',
    ]))

    want_calls.append(
        build_add_binding_call('projects/project1/roles/forsetiBigqueryViewer')
    )

    want_calls.append(mock.call([
        'gcloud', 'iam', 'roles', 'create', 'forsetiCloudsqlViewer',
        '--project', 'project1',
        '--title', 'Forseti CloudSql Metadata Viewer',
        '--description', 'Access to only view CloudSql resources',
        '--stage', 'ALPHA',
        '--permissions',
        ('cloudsql.backupRuns.get,cloudsql.backupRuns.list,'
         'cloudsql.databases.get,cloudsql.databases.list,'
         'cloudsql.instances.get,cloudsql.instances.list,'
         'cloudsql.sslCerts.get,cloudsql.sslCerts.list,cloudsql.users.list'),
    ]))

    want_calls.append(
        build_add_binding_call('projects/project1/roles/forsetiCloudsqlViewer'))

    mock_check_output.assert_has_calls(want_calls)


def build_add_binding_call(role):
  return mock.call([
      'gcloud', 'projects', 'add-iam-policy-binding', 'project1',
      '--member',
      'serviceAccount:forseti-sa@@forseti-project.iam.gserviceaccount.com',
      '--role', role,
  ])

if __name__ == '__main__':
  absltest.main()
