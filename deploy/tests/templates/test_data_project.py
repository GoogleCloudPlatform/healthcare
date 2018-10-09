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

"""Tests for healthcare.deploy.templates.data_project.

These tests check that the template is free from syntax errors and generates
the expected resources.

To run tests, run `python -m unittest tests.data_project_test` from the
templates directory.
"""

import unittest

from templates import data_project


class TestDataProject(unittest.TestCase):

  def test_expansion_local_logging(self):
    class FakeContext(object):
      env = {
          'deployment': 'my-deployment',
          'project': 'my-project',
      }
      properties = {
          'has_organization': True,
          'remove_owner_user': 'some-owner@mydomain.com',
          'owners_group': 'some-admin-group@googlegroups.com',
          'auditors_group': 'some-auditors-group@googlegroups.com',
          'data_readwrite_groups': [
              'some-readwrite-group@googlegroups.com',
              'another-readwrite-group@googlegroups.com',
          ],
          'data_readonly_groups': [
              'some-readonly-group@googlegroups.com',
              'another-readonly-group@googlegroups.com',
          ],
          'local_audit_logs': {
              'logs_gcs_bucket': {
                  'location': 'US',
                  'storage_class': 'MULTI_REGIONAL',
                  'ttl_days': 365,
              },
              'logs_bigquery_dataset': {
                  'location': 'US',
              },
          },
          'bigquery_datasets': [
              {
                  'name': 'us_data',
                  'location': 'US',
              },
              {
                  'name': 'euro_data',
                  'location': 'EU',
              },
          ],
          'data_buckets': [
              {
                  'name_suffix': '-nlp-bucket',
                  'location': 'US-CENTRAL1',
                  'storage_class': 'REGIONAL',
                  'expected_users': [
                      'auth_user_1@mydomain.com',
                      'auth_user_2@mydomain.com',
                  ],
              },
              {
                  'name_suffix': '-other-bucket',
                  'location': 'US-EAST1',
                  'storage_class': 'REGIONAL',
              },
              {
                  'name_suffix': '-euro-bucket',
                  'location': 'EUROPE-WEST1',
                  'storage_class': 'REGIONAL',
                  'expected_users': ['auth_user_3@mydomain.com'],
              },
          ],
          'pubsub': {
              'topic': 'test-topic',
              'subscription': 'test-subscription',
              'publisher_account': (
                  'cloud-healthcare-eng@system.gserviceaccount.com'),
              'ack_deadline_sec': 100
          },
          'enabled_apis': [
              'cloudbuild.googleapis.com',
              'cloudresourcemanager.googleapis.com',  # Ignored by script.
              'containerregistry.googleapis.com',
              'deploymentmanager.googleapis.com',  # Ignored by script.
          ]
      }

    generated = data_project.generate_config(FakeContext())

    expected = {
        'resources': [{
            'name': 'set-project-bindings-get-iam-policy',
            'action': ('gcp-types/cloudresourcemanager-v1:'
                       'cloudresourcemanager.projects.getIamPolicy'),
            'properties': {'resource': 'my-project'},
            'metadata': {'runtimePolicy': ['UPDATE_ALWAYS']},
        }, {
            'name': 'set-project-bindings-patch-iam-policy',
            'action': ('gcp-types/cloudresourcemanager-v1:'
                       'cloudresourcemanager.projects.setIamPolicy'),
            'properties': {
                'resource': 'my-project',
                'policy': '$(ref.set-project-bindings-get-iam-policy)',
                'gcpIamPolicyPatch': {
                    'add': [
                        {
                            'role': 'roles/iam.securityReviewer',
                            'members': [
                                'group:some-auditors-group@googlegroups.com'
                            ],
                        }, {
                            'role': 'roles/owner',
                            'members': [
                                'group:some-admin-group@googlegroups.com'
                            ],
                        }
                    ],
                    'remove': [{
                        'role': 'roles/owner',
                        'members': ['user:some-owner@mydomain.com'],
                    }],
                },
            },
        }, {
            'name': 'my-project-logs',
            'type': 'storage.v1.bucket',
            'accessControl': {
                'gcpIamPolicy': {
                    'bindings': [{
                        'role': 'roles/storage.admin',
                        'members': [
                            'group:some-auditors-group@googlegroups.com'
                        ]
                    }, {
                        'role': 'roles/storage.objectCreator',
                        'members': ['group:cloud-storage-analytics@google.com']
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
            'name': 'audit-logs-to-bigquery',
            'type': 'logging.v2.sink',
            'properties': {
                'sink': 'audit-logs-to-bigquery',
                'uniqueWriterIdentity': True,
                'destination': ('bigquery.googleapis.com/projects/my-project/'
                                'datasets/audit_logs'),
                'filter': 'logName:"logs/cloudaudit.googleapis.com"',
            }
        }, {
            'name': 'create-big-query-dataset-us_data',
            'type': 'bigquery.v2.dataset',
            'properties': {
                'datasetReference': {'datasetId': 'us_data'},
                'location': 'US',
            },
        }, {
            'name': 'update-big-query-dataset-us_data',
            'action': 'gcp-types/bigquery-v2:bigquery.datasets.patch',
            'properties': {
                'projectId': 'my-project',
                'datasetId': 'us_data',
                'access': [{
                    'role': 'OWNER',
                    'groupByEmail': 'some-admin-group@googlegroups.com',
                }, {
                    'role': 'READER',
                    'groupByEmail': 'some-readonly-group@googlegroups.com',
                }, {
                    'role': 'READER',
                    'groupByEmail': 'another-readonly-group@googlegroups.com',
                }, {
                    'role': 'WRITER',
                    'groupByEmail': 'some-readwrite-group@googlegroups.com',
                }, {
                    'role': 'WRITER',
                    'groupByEmail': 'another-readwrite-group@googlegroups.com',
                }],
            },
            'metadata': {'dependsOn': ['create-big-query-dataset-us_data']},
        }, {
            'name': 'create-big-query-dataset-euro_data',
            'type': 'bigquery.v2.dataset',
            'properties': {
                'datasetReference': {'datasetId': 'euro_data'},
                'location': 'EU',
            },
        }, {
            'name': 'update-big-query-dataset-euro_data',
            'action': 'gcp-types/bigquery-v2:bigquery.datasets.patch',
            'properties': {
                'projectId': 'my-project',
                'datasetId': 'euro_data',
                'access': [{
                    'role': 'OWNER',
                    'groupByEmail': 'some-admin-group@googlegroups.com',
                }, {
                    'role': 'READER',
                    'groupByEmail': 'some-readonly-group@googlegroups.com',
                }, {
                    'role': 'READER',
                    'groupByEmail': 'another-readonly-group@googlegroups.com',
                }, {
                    'role': 'WRITER',
                    'groupByEmail': 'some-readwrite-group@googlegroups.com',
                }, {
                    'role': 'WRITER',
                    'groupByEmail': 'another-readwrite-group@googlegroups.com',
                }],
            },
            'metadata': {'dependsOn': ['create-big-query-dataset-euro_data']},
        }, {
            'name': 'my-project-nlp-bucket',
            'type': 'storage.v1.bucket',
            'metadata': {'dependsOn': ['my-project-logs']},
            'accessControl': {
                'gcpIamPolicy': {
                    'bindings': [{
                        'role': 'roles/storage.admin',
                        'members': ['group:some-admin-group@googlegroups.com']
                    }, {
                        'role': 'roles/storage.objectAdmin',
                        'members': [
                            'group:some-readwrite-group@googlegroups.com',
                            'group:another-readwrite-group@googlegroups.com',
                        ]
                    }, {
                        'role': 'roles/storage.objectViewer',
                        'members': [
                            'group:some-readonly-group@googlegroups.com',
                            'group:another-readonly-group@googlegroups.com',
                        ]
                    }]
                }
            },
            'properties': {
                'location': 'US-CENTRAL1',
                'versioning': {'enabled': True},
                'storageClass': 'REGIONAL',
                'logging': {
                    'logBucket': 'my-project-logs'
                }
            }
        }, {
            'name': 'unexpected-access-my-project-nlp-bucket',
            'type': 'logging.v2.metric',
            'properties': {
                'filter': ('resource.type=gcs_bucket AND '
                           'logName=projects/my-project/logs/'
                           'cloudaudit.googleapis.com%2Fdata_access AND '
                           'protoPayload.resourceName=projects/_/buckets/'
                           'my-project-nlp-bucket AND '
                           'protoPayload.authenticationInfo.principalEmail!=('
                           'auth_user_1@mydomain.com AND '
                           'auth_user_2@mydomain.com)'),
                'description':
                    'Count of unexpected data access to my-project-nlp-bucket.',
                'labelExtractors': {
                    'user': ('EXTRACT('
                             'protoPayload.authenticationInfo.principalEmail)'),
                },
                'metricDescriptor': {
                    'labels': [{
                        'description': 'Unexpected user',
                        'key': 'user',
                        'valueType': 'STRING'
                    }],
                    'unit': '1',
                    'metricKind': 'DELTA',
                    'valueType': 'INT64'
                },
                'metric': 'unexpected-access-my-project-nlp-bucket'
            }
        }, {
            'name': 'my-project-other-bucket',
            'type': 'storage.v1.bucket',
            'metadata': {'dependsOn': ['my-project-logs']},
            'accessControl': {
                'gcpIamPolicy': {
                    'bindings': [{
                        'role': 'roles/storage.admin',
                        'members': ['group:some-admin-group@googlegroups.com']
                    }, {
                        'role': 'roles/storage.objectAdmin',
                        'members': [
                            'group:some-readwrite-group@googlegroups.com',
                            'group:another-readwrite-group@googlegroups.com',
                        ]
                    }, {
                        'role': 'roles/storage.objectViewer',
                        'members': [
                            'group:some-readonly-group@googlegroups.com',
                            'group:another-readonly-group@googlegroups.com',
                        ]
                    }]
                }
            },
            'properties': {
                'location': 'US-EAST1',
                'versioning': {'enabled': True},
                'storageClass': 'REGIONAL',
                'logging': {
                    'logBucket': 'my-project-logs'
                }
            }
        }, {
            'name': 'my-project-euro-bucket',
            'type': 'storage.v1.bucket',
            'metadata': {'dependsOn': ['my-project-logs']},
            'accessControl': {
                'gcpIamPolicy': {
                    'bindings': [{
                        'role': 'roles/storage.admin',
                        'members': ['group:some-admin-group@googlegroups.com']
                    }, {
                        'role': 'roles/storage.objectAdmin',
                        'members': [
                            'group:some-readwrite-group@googlegroups.com',
                            'group:another-readwrite-group@googlegroups.com',
                        ]
                    }, {
                        'role': 'roles/storage.objectViewer',
                        'members': [
                            'group:some-readonly-group@googlegroups.com',
                            'group:another-readonly-group@googlegroups.com',
                        ]
                    }]
                }
            },
            'properties': {
                'location': 'EUROPE-WEST1',
                'versioning': {'enabled': True},
                'storageClass': 'REGIONAL',
                'logging': {
                    'logBucket': 'my-project-logs'
                }
            }
        }, {
            'name': 'unexpected-access-my-project-euro-bucket',
            'type': 'logging.v2.metric',
            'properties': {
                'filter': ('resource.type=gcs_bucket AND '
                           'logName=projects/my-project/logs/'
                           'cloudaudit.googleapis.com%2Fdata_access AND '
                           'protoPayload.resourceName=projects/_/buckets/'
                           'my-project-euro-bucket AND '
                           'protoPayload.authenticationInfo.principalEmail!=('
                           'auth_user_3@mydomain.com)'),
                'description': ('Count of unexpected data access to '
                                'my-project-euro-bucket.'),
                'labelExtractors': {
                    'user': ('EXTRACT('
                             'protoPayload.authenticationInfo.principalEmail)'),
                },
                'metricDescriptor': {
                    'labels': [{
                        'description': 'Unexpected user',
                        'key': 'user',
                        'valueType': 'STRING'
                    }],
                    'unit': '1',
                    'metricKind': 'DELTA',
                    'valueType': 'INT64'
                },
                'metric': 'unexpected-access-my-project-euro-bucket'
            }
        }, {
            'name': 'test-topic',
            'type': 'pubsub.v1.topic',
            'properties': {
                'topic': 'test-topic',
            },
            'accessControl': {
                'gcpIamPolicy': {
                    'bindings': [
                        {
                            'role': 'roles/pubsub.publisher',
                            'members': [
                                ('serviceAccount:cloud-healthcare-eng'
                                 '@system.gserviceaccount.com')
                            ],
                        },
                    ],
                },
            },
        }, {
            'name': 'test-subscription',
            'type': 'pubsub.v1.subscription',
            'properties': {
                'subscription': 'test-subscription',
                'topic': 'projects/my-project/topics/test-topic',
                'ackDeadlineSeconds': 100,
            },
            'accessControl': {
                'gcpIamPolicy': {
                    'bindings': [{
                        'role':
                            'roles/pubsub.editor',
                        'members': [
                            'group:some-readwrite-group@googlegroups.com',
                            ('group:another-readwrite-group@googlegroups.'
                             'com'),
                        ],
                    },],
                },
            },
            'metadata': {
                'dependsOn': ['test-topic'],
            },
        }, {
            'name': 'iam-policy-change-count',
            'type': 'logging.v2.metric',
            'properties': {
                'filter': ('\n'
                           '      resource.type=project AND\n'
                           '      protoPayload.serviceName='
                           'cloudresourcemanager.googleapis.com AND\n'
                           '      protoPayload.methodName=SetIamPolicy'),
                'description': 'Count of IAM policy changes.',
                'labelExtractors': {
                    'user': ('EXTRACT('
                             'protoPayload.authenticationInfo.principalEmail)'),
                },
                'metricDescriptor': {
                    'labels': [{
                        'description': 'Unexpected user',
                        'key': 'user',
                        'valueType': 'STRING'
                    }],
                    'unit': '1',
                    'metricKind': 'DELTA',
                    'valueType': 'INT64'
                },
                'metric': 'iam-policy-change-count'
            }
        }, {
            'name': 'bucket-permission-change-count',
            'type': 'logging.v2.metric',
            'properties': {
                'filter': (
                    '\n'
                    '      resource.type=gcs_bucket AND\n'
                    '      protoPayload.serviceName=storage.googleapis.com '
                    'AND\n'
                    '      (protoPayload.methodName=storage.setIamPermissions '
                    'OR\n'
                    '       protoPayload.methodName=storage.objects.update)'),
                'description':
                    'Count of GCS permissions changes.',
                'labelExtractors': {
                    'user': ('EXTRACT('
                             'protoPayload.authenticationInfo.principalEmail)'),
                },
                'metricDescriptor': {
                    'labels': [{
                        'description': 'Unexpected user',
                        'key': 'user',
                        'valueType': 'STRING'
                    }],
                    'unit':
                        '1',
                    'metricKind':
                        'DELTA',
                    'valueType':
                        'INT64'
                },
                'metric':
                    'bucket-permission-change-count'
            }
        }, {
            'name': 'audit-configs-get-iam-etag',
            'action': ('gcp-types/cloudresourcemanager-v1:'
                       'cloudresourcemanager.projects.getIamPolicy'),
            'properties': {
                'resource': 'my-project',
            },
            'metadata': {
                'runtimePolicy': ['UPDATE_ALWAYS'],
                'dependsOn': ['set-project-bindings-patch-iam-policy'],
            },
        }, {
            'name': 'audit-configs-patch-iam-policy',
            'action': ('gcp-types/cloudresourcemanager-v1:'
                       'cloudresourcemanager.projects.setIamPolicy'),
            'properties': {
                'updateMask': 'auditConfigs,etag',
                'resource': 'my-project',
                'policy': {
                    'auditConfigs': [{
                        'auditLogConfigs': [
                            {'logType': 'ADMIN_READ'},
                            {'logType': 'DATA_WRITE'},
                            {'logType': 'DATA_READ'},
                        ],
                        'service': 'allServices',
                    }],
                    'etag': '$(ref.audit-configs-get-iam-etag.etag)',
                },
            },
            'metadata': {
                'dependsOn': ['audit-configs-get-iam-etag'],
            },
        }, {
            'name': 'enable-cloudbuild',
            'action': ('gcp-types/servicemanagement-v1:'
                       'servicemanagement.services.enable'),
            'properties': {
                'consumerId': 'project:my-project',
                'serviceName': 'cloudbuild.googleapis.com'
            },
            'metadata': {
                'dependsOn': ['audit-configs-patch-iam-policy'],
            },
        }, {
            'name': 'enable-containerregistry',
            'action': ('gcp-types/servicemanagement-v1:'
                       'servicemanagement.services.enable'),
            'properties': {
                'consumerId': 'project:my-project',
                'serviceName': 'containerregistry.googleapis.com'
            },
            'metadata': {
                'dependsOn': ['audit-configs-patch-iam-policy'],
            },
        }]
    }

    self.assertEqual(generated, expected)

  def testExpansionRemoteLoggingNoOrg(self):
    class FakeContext(object):
      env = {
          'deployment': 'my-deployment',
          'project': 'my-project',
      }
      properties = {
          'has_organization': False,
          'owners_group': 'some-admin-group@googlegroups.com',
          'editors_group': 'some-editors-group@googlegroups.com',
          'auditors_group': 'some-auditors-group@googlegroups.com',
          'additional_project_permissions': [
              {
                  'roles': ['roles/editor',],
                  'members': ['serviceAccount:s1234@service.accounts',
                              'serviceAccount:s5678@service.accounts']
              }, {
                  'roles': ['roles/bigquery.dataViewer',
                            'roles/storage.objectViewer'],
                  'members': ['group:extra-viewers@googlegroups.com',
                              'user:some-viewer@mydomain.com']
              },
          ],
          'data_readwrite_groups': ['some-readwrite-group@googlegroups.com'],
          'data_readonly_groups': ['some-readonly-group@googlegroups.com'],
          'remote_audit_logs': {
              'audit_logs_project_id': 'my-audit-logs',
              'logs_gcs_bucket_name': 'some_remote_bucket',
              'logs_bigquery_dataset_id': 'some_remote_dataset',
          },
          'bigquery_datasets': [
              {
                  'name': 'us_data',
                  'location': 'US',
              },
          ],
          'data_buckets': [
              {
                  'name_suffix': '-data',
                  'location': 'US',
                  'storage_class': 'MULTI_REGIONAL',
              },
          ],
      }

    generated = data_project.generate_config(FakeContext())

    expected = {
        'resources': [{
            'name': 'set-project-bindings-get-iam-policy',
            'action': ('gcp-types/cloudresourcemanager-v1:'
                       'cloudresourcemanager.projects.getIamPolicy'),
            'properties': {'resource': 'my-project'},
            'metadata': {'runtimePolicy': ['UPDATE_ALWAYS']},
        }, {
            'name': 'set-project-bindings-patch-iam-policy',
            'action': ('gcp-types/cloudresourcemanager-v1:'
                       'cloudresourcemanager.projects.setIamPolicy'),
            'properties': {
                'resource': 'my-project',
                'policy': '$(ref.set-project-bindings-get-iam-policy)',
                'gcpIamPolicyPatch': {
                    'add': [
                        {
                            'role': 'roles/bigquery.dataViewer',
                            'members': [
                                'group:extra-viewers@googlegroups.com',
                                'user:some-viewer@mydomain.com',
                            ],
                        }, {
                            'role': 'roles/editor',
                            'members': [
                                'group:some-editors-group@googlegroups.com',
                                'serviceAccount:s1234@service.accounts',
                                'serviceAccount:s5678@service.accounts',
                            ],
                        }, {
                            'role': 'roles/iam.securityReviewer',
                            'members': [
                                'group:some-auditors-group@googlegroups.com',
                            ],
                        }, {
                            'role': 'roles/resourcemanager.projectIamAdmin',
                            'members': [
                                'group:some-admin-group@googlegroups.com',
                            ],
                        }, {
                            'role': 'roles/storage.objectViewer',
                            'members': [
                                'group:extra-viewers@googlegroups.com',
                                'user:some-viewer@mydomain.com',
                            ],
                        },
                    ],
                },
            },
        }, {
            'name': 'audit-logs-to-bigquery',
            'type': 'logging.v2.sink',
            'properties': {
                'sink': 'audit-logs-to-bigquery',
                'uniqueWriterIdentity': True,
                'destination': ('bigquery.googleapis.com/projects/'
                                'my-audit-logs/datasets/some_remote_dataset'),
                'filter': 'logName:"logs/cloudaudit.googleapis.com"',
            }
        }, {
            'name': 'create-big-query-dataset-us_data',
            'type': 'bigquery.v2.dataset',
            'properties': {
                'datasetReference': {'datasetId': 'us_data'},
                'location': 'US',
            },
        }, {
            'name': 'update-big-query-dataset-us_data',
            'action': 'gcp-types/bigquery-v2:bigquery.datasets.patch',
            'properties': {
                'projectId': 'my-project',
                'datasetId': 'us_data',
                'access': [{
                    'role': 'OWNER',
                    'groupByEmail': 'some-admin-group@googlegroups.com',
                }, {
                    'role': 'READER',
                    'groupByEmail': 'some-readonly-group@googlegroups.com',
                }, {
                    'role': 'WRITER',
                    'groupByEmail': 'some-readwrite-group@googlegroups.com',
                }],
            },
            'metadata': {'dependsOn': ['create-big-query-dataset-us_data']},
        }, {
            'name': 'my-project-data',
            'type': 'storage.v1.bucket',
            'accessControl': {
                'gcpIamPolicy': {
                    'bindings': [{
                        'role': 'roles/storage.admin',
                        'members': ['group:some-admin-group@googlegroups.com']
                    }, {
                        'role': 'roles/storage.objectAdmin',
                        'members': [
                            'group:some-readwrite-group@googlegroups.com'
                        ]
                    }, {
                        'role': 'roles/storage.objectViewer',
                        'members': [
                            'group:some-readonly-group@googlegroups.com'
                        ]
                    }]
                }
            },
            'properties': {
                'location': 'US',
                'versioning': {'enabled': True},
                'storageClass': 'MULTI_REGIONAL',
                'logging': {
                    'logBucket': 'some_remote_bucket'
                }
            },
        }, {
            'name': 'iam-policy-change-count',
            'type': 'logging.v2.metric',
            'properties': {
                'filter': ('\n'
                           '      resource.type=project AND\n'
                           '      protoPayload.serviceName='
                           'cloudresourcemanager.googleapis.com AND\n'
                           '      protoPayload.methodName=SetIamPolicy'),
                'description': 'Count of IAM policy changes.',
                'labelExtractors': {
                    'user': ('EXTRACT('
                             'protoPayload.authenticationInfo.principalEmail)'),
                },
                'metricDescriptor': {
                    'labels': [{
                        'description': 'Unexpected user',
                        'key': 'user',
                        'valueType': 'STRING'
                    }],
                    'unit': '1',
                    'metricKind': 'DELTA',
                    'valueType': 'INT64'
                },
                'metric': 'iam-policy-change-count'
            }
        }, {
            'name': 'bucket-permission-change-count',
            'type': 'logging.v2.metric',
            'properties': {
                'filter': (
                    '\n'
                    '      resource.type=gcs_bucket AND\n'
                    '      protoPayload.serviceName=storage.googleapis.com '
                    'AND\n'
                    '      (protoPayload.methodName=storage.setIamPermissions '
                    'OR\n'
                    '       protoPayload.methodName=storage.objects.update)'),
                'description':
                    'Count of GCS permissions changes.',
                'labelExtractors': {
                    'user': ('EXTRACT('
                             'protoPayload.authenticationInfo.principalEmail)'),
                },
                'metricDescriptor': {
                    'labels': [{
                        'description': 'Unexpected user',
                        'key': 'user',
                        'valueType': 'STRING'
                    }],
                    'unit':
                        '1',
                    'metricKind':
                        'DELTA',
                    'valueType':
                        'INT64'
                },
                'metric':
                    'bucket-permission-change-count'
            }
        }, {
            'name': 'audit-configs-get-iam-etag',
            'action': ('gcp-types/cloudresourcemanager-v1:'
                       'cloudresourcemanager.projects.getIamPolicy'),
            'properties': {
                'resource': 'my-project',
            },
            'metadata': {
                'runtimePolicy': ['UPDATE_ALWAYS'],
                'dependsOn': ['set-project-bindings-patch-iam-policy'],
            },
        }, {
            'name': 'audit-configs-patch-iam-policy',
            'action': ('gcp-types/cloudresourcemanager-v1:'
                       'cloudresourcemanager.projects.setIamPolicy'),
            'properties': {
                'updateMask': 'auditConfigs,etag',
                'resource': 'my-project',
                'policy': {
                    'auditConfigs': [{
                        'auditLogConfigs': [
                            {'logType': 'ADMIN_READ'},
                            {'logType': 'DATA_WRITE'},
                            {'logType': 'DATA_READ'},
                        ],
                        'service': 'allServices',
                    }],
                    'etag': '$(ref.audit-configs-get-iam-etag.etag)',
                },
            },
            'metadata': {
                'dependsOn': ['audit-configs-get-iam-etag'],
            },
        }]
    }

    self.assertEqual(generated, expected)


if __name__ == '__main__':
  unittest.main()
