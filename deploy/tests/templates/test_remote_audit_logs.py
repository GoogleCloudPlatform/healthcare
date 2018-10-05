# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Tests for healthcare.deploy.templates.remote_audit_logs.

These tests check that the template is free from syntax errors and generates
the expected resources.

To run tests, run `python -m unittest tests.remote_audit_logs_test` from the
templates directory.
"""

import unittest

from templates import remote_audit_logs


class TestRemoteAuditLogsTemplate(unittest.TestCase):

  def test_template_expansion(self):
    class FakeContext(object):
      env = {
          'deployment': 'my-deployment',
          'project': 'my-audit-logs',
      }
      properties = {
          'owners_group': 'my-audit-logs-owners@googlegroups.com',
          'auditors_group': 'my-data-auditors@googlegroups.com',
          'logs_gcs_bucket': {
              'name': 'my-audit-logs-my-project',
              'location': 'US',
              'storage_class': 'MULTI_REGIONAL',
              'ttl_days': 365,
          },
          'logs_bigquery_dataset': {
              'name': 'my_project',
              'location': 'US',
              'log_sink_service_account': (
                  'audit-logs-to-bigquery@'
                  'logging-12345.iam.gserviceaccount.com'),
          },
      }

    generated = remote_audit_logs.GenerateConfig(FakeContext())

    expected = {
        'resources': [{
            'name': 'my-audit-logs-my-project',
            'type': 'storage.v1.bucket',
            'accessControl': {
                'gcpIamPolicy': {
                    'bindings': [{
                        'role': 'roles/storage.admin',
                        'members': [
                            'group:my-audit-logs-owners@googlegroups.com'
                        ]
                    }, {
                        'role': 'roles/storage.objectCreator',
                        'members': ['group:cloud-storage-analytics@google.com']
                    }, {
                        'role': 'roles/storage.objectViewer',
                        'members': ['group:my-data-auditors@googlegroups.com']
                    }]
                }
            },
            'properties': {
                'location': 'US',
                'storageClass': 'MULTI_REGIONAL',
                'lifecycle': {
                    'rule': [{
                        'action': {'type': 'Delete'},
                        'condition': {
                            'isLive': True,
                            'age': 365
                        }
                    }]
                }
            }
        }, {
            'name': 'my_project',
            'type': 'bigquery.v2.dataset',
            'properties': {
                'datasetReference': {'datasetId': 'my_project'},
                'location': 'US',
            },
        }, {
            'name': 'update-my_project',
            'action': 'gcp-types/bigquery-v2:bigquery.datasets.patch',
            'properties': {
                'projectId': 'my-audit-logs',
                'datasetId': 'my_project',
                'access': [{
                    'role': 'OWNER',
                    'groupByEmail': 'my-audit-logs-owners@googlegroups.com',
                }, {
                    'role': 'READER',
                    'groupByEmail': 'my-data-auditors@googlegroups.com',
                }, {
                    'role': 'WRITER',
                    'userByEmail': ('audit-logs-to-bigquery@'
                                    'logging-12345.iam.gserviceaccount.com'),
                }],
            },
            'metadata': {'dependsOn': ['my_project']},
        }]
    }

    self.assertEqual(generated, expected)


if __name__ == '__main__':
  unittest.main()
