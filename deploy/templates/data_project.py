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
"""Configures a data project with storage and logging.

For details and usage, see deploy/README.md.
"""

# Map from prefixes used by IAM to those used in BigQuery.
_IAM_TO_BIGQUERY_MEMBER = {
    'user': 'userByEmail',
    'group': 'groupByEmail',
    'domain': 'domain',
    # BigQuery treats serviceAccounts as users.
    'serviceAccount': 'userByEmail',
}

DEFAULT_CUSTOM_ROLE_STAGE = 'GA'


def _get_bigquery_access_for_role(role_name, members):
  """Converts role and IAM style members to BigQuery style ACL."""

  access_for_role = []
  for member in members:
    if member == 'allAuthenticatedUsers':
      access_for_role.append({'role': role_name, 'specialGroup': member})
    else:
      member_type, member_name = member.split(':')
      access_for_role.append({
          'role': role_name,
          _IAM_TO_BIGQUERY_MEMBER[member_type]: member_name
      })
  return access_for_role


def generate_config(context):
  """Generate Deployment Manager configuration."""

  project_id = context.env['project']

  if ('local_audit_logs' in context.properties) == (
      'remote_audit_logs' in context.properties):
    raise ValueError('Must specify local_audit_logs or remote_audit_logs but '
                     'not both.')
  use_local_logs = 'local_audit_logs' in context.properties
  has_organization = context.properties['has_organization']

  resources = []

  # Custom roles
  custom_roles = context.properties.get('custom_roles', [])
  for role in custom_roles:
    name = role['name']
    permissions = role['permissions']
    description = role.get('description', name)
    title = role.get('title', name)
    resources.append({
        'name': name,
        'type': 'gcp-types/iam-v1:projects.roles',
        'properties': {
            'parent': 'projects/' + project_id,
            'roleId': name,
            'role': {
                'title': title,
                'description': description,
                'stage': DEFAULT_CUSTOM_ROLE_STAGE,
                'includedPermissions': permissions,
            }
        }
    })

  # Set project-level IAM roles. Adding owners and auditors roles, and removing
  # the single-owner. Non-organization projects cannot have a owner group, so
  # use projectIamAdmin instead.
  if has_organization:
    owners_group_role = 'roles/owner'
  else:
    owners_group_role = 'roles/resourcemanager.projectIamAdmin'

  project_bindings = {
      owners_group_role: ['group:' + context.properties['owners_group']],
      'roles/iam.securityReviewer': [
          'group:' + context.properties['auditors_group']
      ],
  }
  if 'editors_group' in context.properties:
    project_bindings['roles/editor'] = [
        'group:' + context.properties['editors_group']
    ]

  # Merge in additional permissions, which may include the above roles.
  for additional in context.properties.get('additional_project_permissions',
                                           []):
    for role in additional['roles']:
      project_bindings[role] = (
          project_bindings.get(role, []) + additional['members'])

  policy_patch = {
      'add': [{
          'role': role,
          'members': members
      } for role, members in sorted(project_bindings.items())]
  }
  if has_organization and 'remove_owner_user' in context.properties:
    policy_patch['remove'] = [{
        'role': 'roles/owner',
        'members': ['user:' + context.properties['remove_owner_user']],
    }]
  get_iam_policy_name = 'set-project-bindings-get-iam-policy'
  resources.extend([
      {
          'name': get_iam_policy_name,
          'action': ('gcp-types/cloudresourcemanager-v1:'
                     'cloudresourcemanager.projects.getIamPolicy'),
          'properties': {
              'resource': project_id,
          },
          'metadata': {
              'runtimePolicy': ['UPDATE_ALWAYS'],
          },
      },
      {
          'name': 'set-project-bindings-patch-iam-policy',
          'action': ('gcp-types/cloudresourcemanager-v1:'
                     'cloudresourcemanager.projects.setIamPolicy'),
          'properties': {
              'resource': project_id,
              'policy': '$(ref.' + get_iam_policy_name + ')',
              'gcpIamPolicyPatch': policy_patch,
          },
          'metadata': {
              'runtimePolicy': ['UPDATE_ON_CHANGE'],
          },
      },
  ])

  # Create a logs GCS bucket and BigQuery dataset, or get the names of the
  # remote bucket and dataset.
  previous_gcs_bucket = None
  logs_bucket_id = None
  if use_local_logs:
    logs_gcs_bucket = context.properties['local_audit_logs'].get(
        'logs_gcs_bucket')
    # Logs GCS bucket is only needed if there are data GCS buckets.
    if logs_gcs_bucket:
      logs_bucket_id = project_id + '-logs'
      # Create the local GCS bucket to hold logs.
      resources.append({
          'name': logs_bucket_id,
          'type': 'storage.v1.bucket',
          'properties': {
              'location': logs_gcs_bucket['location'],
              'storageClass': logs_gcs_bucket['storage_class'],
              'lifecycle': {
                  'rule': [{
                      'action': {
                          'type': 'Delete'
                      },
                      'condition': {
                          'age': logs_gcs_bucket['ttl_days'],
                          'isLive': True,
                      },
                  }],
              },
          },
          'accessControl': {
              'gcpIamPolicy': {
                  'bindings': [
                      {
                          'role':
                              'roles/storage.admin',
                          'members': [
                              'group:' + context.properties['owners_group']
                          ],
                      },
                      {
                          'role':
                              'roles/storage.objectViewer',
                          'members': [
                              'group:' + context.properties['auditors_group']
                          ],
                      },
                      {
                          'role':
                              'roles/storage.objectCreator',
                          'members': [
                              'group:cloud-storage-analytics@google.com'
                          ],
                      },
                  ],
              },
          },
      })
      previous_gcs_bucket = logs_bucket_id

    # Get name of local BigQuery dataset to hold audit logs.
    # This dataset will need to be created after running this deployment
    dataset_id = 'audit_logs'
    log_sink_destination = ('bigquery.googleapis.com/projects/' + project_id +
                            '/datasets/' + dataset_id)
  else:
    logs_bucket_id = context.properties['remote_audit_logs'].get(
        'logs_gcs_bucket_name')

    log_sink_destination = (
        'bigquery.googleapis.com/projects/' +
        context.properties['remote_audit_logs']['audit_logs_project_id'] +
        '/datasets/' +
        context.properties['remote_audit_logs']['logs_bigquery_dataset_id'])

  # Create a logs metric sink of audit logs to a BigQuery dataset. This also
  # creates a service account that must be given WRITER access to the dataset.
  log_sink_name = 'audit-logs-to-bigquery'
  resources.append({
      'name': log_sink_name,
      'type': 'logging.v2.sink',
      'properties': {
          'sink': log_sink_name,
          'destination': log_sink_destination,
          'filter': 'logName:"logs/cloudaudit.googleapis.com"',
          'uniqueWriterIdentity': True,
      },
  })

  # BigQuery dataset(s) to hold actual data. Create serially to avoid exceeding
  # API quota.
  previous_bq_update = None
  for bq_dataset in context.properties.get('bigquery_datasets', []):
    ds_name = bq_dataset['name']
    bq_create_resource = {
        'name': 'create-big-query-dataset-' + ds_name,
        'type': 'bigquery.v2.dataset',
        'properties': {
            'datasetReference': {
                'datasetId': ds_name,
            },
            'location': bq_dataset['location'],
        },
    }
    if previous_bq_update:
      bq_create_resource['metadata'] = {'dependsOn': [previous_bq_update]}
    resources.append(bq_create_resource)

    add_permissions = bq_dataset.get('additional_dataset_permissions', {})
    access_list = [{
        'role': 'OWNER',
        'groupByEmail': context.properties['owners_group']
    }]

    for reader in context.properties.get('data_readonly_groups', []):
      access_list.append({'role': 'READER', 'groupByEmail': reader})

    for writer in context.properties.get('data_readwrite_groups', []):
      access_list.append({'role': 'WRITER', 'groupByEmail': writer})

    access_list += (
        _get_bigquery_access_for_role('OWNER', add_permissions.get(
            'owners', [])) + _get_bigquery_access_for_role(
                'WRITER', add_permissions.get('readwrite', [])) +
        _get_bigquery_access_for_role('READER',
                                      add_permissions.get('readonly', [])))

    # Update permissions for the dataset. This also removes the deployment
    # manager service account's access.
    previous_bq_update = 'update-big-query-dataset-' + ds_name
    resources.append({
        'name': previous_bq_update,
        'action': 'gcp-types/bigquery-v2:bigquery.datasets.patch',
        'properties': {
            'projectId': project_id,
            'datasetId': ds_name,
            'access': access_list,
        },
        'metadata': {
            'dependsOn': ['create-big-query-dataset-' + ds_name],
            'runtimePolicy': ['UPDATE_ON_CHANGE'],
        },
    })

  # GCS bucket(s) to hold actual data. Create serially to avoid exceeding API
  # quota.

  default_bucket_owners = ['group:' + context.properties['owners_group']]
  default_bucket_readwrite = [
      'group:' + readwrite
      for readwrite in context.properties.get('data_readwrite_groups', [])
  ]
  default_bucket_readonly = [
      'group:' + readonly
      for readonly in context.properties.get('data_readonly_groups', [])
  ]

  for data_bucket in context.properties.get('data_buckets', []):
    if not logs_bucket_id:
      raise ValueError('Logs GCS bucket must be provided for data buckets.')

    bucket_roles = []
    add_permissions = data_bucket.get('additional_bucket_permissions', {})
    bucket_roles.append({
        'role': 'roles/storage.admin',
        'members': default_bucket_owners + add_permissions.get('owners', [])
    })
    bucket_roles.append({
        'role':
            'roles/storage.objectAdmin',
        'members':
            default_bucket_readwrite + add_permissions.get('readwrite', [])
    })
    bucket_roles.append({
        'role': 'roles/storage.objectViewer',
        'members': default_bucket_readonly + add_permissions.get(
            'readonly', [])
    })
    bucket_roles.append({
        'role': 'roles/storage.objectCreator',
        'members': add_permissions.get('writeonly', [])
    })

    bindings = [role for role in bucket_roles if role['members']]

    data_bucket_id = project_id + data_bucket['name_suffix']
    data_bucket_resource = {
        'name': data_bucket_id,
        'type': 'storage.v1.bucket',
        'properties': {
            'location': data_bucket['location'],
            'storageClass': data_bucket['storage_class'],
            'logging': {
                'logBucket': logs_bucket_id,
            },
            'versioning': {
                'enabled': True,
            },
        },
        'accessControl': {
            'gcpIamPolicy': {
                'bindings': bindings,
            },
        },
    }
    if previous_gcs_bucket:
      data_bucket_resource['metadata'] = {
          'dependsOn': [previous_gcs_bucket],
      }
    resources.append(data_bucket_resource)
    previous_gcs_bucket = data_bucket_id

    # Create a logs-based metric for unexpected users, if a list of expected
    # users is provided.
    if 'expected_users' in data_bucket:
      unexpected_access_filter = (
          'resource.type=gcs_bucket AND '
          'logName=projects/%(project_id)s/logs/'
          'cloudaudit.googleapis.com%%2Fdata_access AND '
          'protoPayload.resourceName=projects/_/buckets/%(bucket_id)s AND '
          'protoPayload.authenticationInfo.principalEmail!=(%(exp_users)s)') % {
              'project_id': project_id,
              'bucket_id': data_bucket_id,
              'exp_users': (' AND '.join(data_bucket['expected_users']))
          }
      resources.append({
          'name': 'unexpected-access-' + data_bucket_id,
          'type': 'logging.v2.metric',
          'properties': {
              'metric':
                  'unexpected-access-' + data_bucket_id,
              'description':
                  'Count of unexpected data access to ' + data_bucket_id + '.',
              'filter':
                  unexpected_access_filter,
              'metricDescriptor': {
                  'metricKind':
                      'DELTA',
                  'valueType':
                      'INT64',
                  'unit':
                      '1',
                  'labels': [{
                      'key': 'user',
                      'valueType': 'STRING',
                      'description': 'Unexpected user',
                  }],
              },
              'labelExtractors': {
                  'user':
                      'EXTRACT(protoPayload.authenticationInfo.principalEmail)'
              },
          },
      })

  # Create a Pub/Sub topic for the Cloud Healthcare service account to publish
  # to, with a subscription for the readwrite group.
  if 'pubsub' in context.properties:
    pubsub_config = context.properties['pubsub']
    topic_name = pubsub_config['topic']
    publisher_account = pubsub_config['publisher_account']
    resources.append({
        'name': topic_name,
        'type': 'pubsub.v1.topic',
        'properties': {
            'topic': topic_name,
        },
        'accessControl': {
            'gcpIamPolicy': {
                'bindings': [{
                    'role': 'roles/pubsub.publisher',
                    'members': ['serviceAccount:' + publisher_account],
                }],
            },
        },
    })
    resources.append({
        'name': pubsub_config['subscription'],
        'type': 'pubsub.v1.subscription',
        'properties': {
            'subscription': pubsub_config['subscription'],
            'topic': 'projects/{}/topics/{}'.format(project_id, topic_name),
            'ackDeadlineSeconds': pubsub_config['ack_deadline_sec']
        },
        'accessControl': {
            'gcpIamPolicy': {
                'bindings': [{
                    'role':
                        'roles/pubsub.editor',
                    'members': [
                        'group:' + writer for writer in
                        context.properties['data_readwrite_groups']
                    ],
                },],
            },
        },
        'metadata': {
            'dependsOn': [topic_name],
        },
    })

  # Create Logs-based metrics for IAM policy changes.
  policy_change_filter = ('protoPayload.methodName="SetIamPolicy" OR\n'
                          'protoPayload.methodName:".setIamPolicy"')
  resources.append({
      'name': 'iam-policy-change-count',
      'type': 'logging.v2.metric',
      'properties': {
          'metric': 'iam-policy-change-count',
          'description': 'Count of IAM policy changes.',
          'filter': policy_change_filter,
          'metricDescriptor': {
              'metricKind':
                  'DELTA',
              'valueType':
                  'INT64',
              'unit':
                  '1',
              'labels': [{
                  'key': 'user',
                  'valueType': 'STRING',
                  'description': 'Unexpected user',
              }],
          },
          'labelExtractors': {
              'user': 'EXTRACT(protoPayload.authenticationInfo.principalEmail)'
          },
      },
  })

  # Create Logs-based metrics for GCS bucket permission changes.
  bucket_change_filter = """
      resource.type=gcs_bucket AND
      protoPayload.serviceName=storage.googleapis.com AND
      (protoPayload.methodName=storage.setIamPermissions OR
       protoPayload.methodName=storage.objects.update)"""
  resources.append({
      'name': 'bucket-permission-change-count',
      'type': 'logging.v2.metric',
      'properties': {
          'metric': 'bucket-permission-change-count',
          'description': 'Count of GCS permissions changes.',
          'filter': bucket_change_filter,
          'metricDescriptor': {
              'metricKind':
                  'DELTA',
              'valueType':
                  'INT64',
              'unit':
                  '1',
              'labels': [{
                  'key': 'user',
                  'valueType': 'STRING',
                  'description': 'Unexpected user',
              }],
          },
          'labelExtractors': {
              'user': 'EXTRACT(protoPayload.authenticationInfo.principalEmail)'
          },
      },
  })

  # Create Logs-based metrics for Bigquery permission changes.
  bigquery_change_filter = ('resource.type="bigquery_resource" AND\n'
                            'protoPayload.methodName="datasetservice.update"')
  resources.append({
      'name': 'bigquery-settings-change-count',
      'type': 'logging.v2.metric',
      'properties': {
          'metric': 'bigquery-settings-change-count',
          'description': 'Count of bigquery permission changes.',
          'filter': bigquery_change_filter,
          'metricDescriptor': {
              'metricKind':
                  'DELTA',
              'valueType':
                  'INT64',
              'unit':
                  '1',
              'labels': [{
                  'key': 'user',
                  'valueType': 'STRING',
                  'description': 'Unexpected user',
              }],
          },
          'labelExtractors': {
              'user': 'EXTRACT(protoPayload.authenticationInfo.principalEmail)'
          },
      },
  })

  # Enable data-access logging. UPDATE_ALWAYS is added to metadata to get a new
  # etag each time.
  resources.extend([
      {
          'name': 'audit-configs-get-iam-etag',
          'action': ('gcp-types/cloudresourcemanager-v1:'
                     'cloudresourcemanager.projects.getIamPolicy'),
          'properties': {
              'resource': project_id,
          },
          'metadata': {
              'dependsOn': ['set-project-bindings-patch-iam-policy'],
              'runtimePolicy': ['UPDATE_ALWAYS'],
          },
      },
      {
          'name': 'audit-configs-patch-iam-policy',
          'action': ('gcp-types/cloudresourcemanager-v1:'
                     'cloudresourcemanager.projects.setIamPolicy'),
          'properties': {
              'resource': project_id,
              'policy': {
                  'etag':
                      '$(ref.audit-configs-get-iam-etag.etag)',
                  'auditConfigs': [{
                      'auditLogConfigs': [
                          {
                              'logType': 'ADMIN_READ'
                          },
                          {
                              'logType': 'DATA_WRITE'
                          },
                          {
                              'logType': 'DATA_READ'
                          },
                      ],
                      'service': 'allServices',
                  }],
              },
              'updateMask': 'auditConfigs,etag',
          },
          'metadata': {
              'dependsOn': ['audit-configs-get-iam-etag'],
              'runtimePolicy': ['UPDATE_ON_CHANGE'],
          },
      },
  ])

  return {'resources': resources}
