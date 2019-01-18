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
"""Creates new GCS buckets and BigQuery datasets in an Audit Logs project."""


def generate_config(context):
  """Generate Deployment Manager configuration."""

  project_id = context.env['project']
  owners_group = context.properties['owners_group']
  auditors_group = context.properties['auditors_group']
  resources = []

  # The GCS bucket to hold logs.
  logs_bucket = context.properties.get('logs_gcs_bucket')
  if logs_bucket:
    resources.append({
        'name': logs_bucket['name'],
        'type': 'storage.v1.bucket',
        'properties': {
            'location': logs_bucket['location'],
            'storageClass': logs_bucket['storage_class'],
            'lifecycle': {
                'rule': [{
                    'action': {
                        'type': 'Delete'
                    },
                    'condition': {
                        'age': logs_bucket['ttl_days'],
                        'isLive': True,
                    },
                }],
            },
        },
        'accessControl': {
            'gcpIamPolicy': {
                'bindings': [
                    {
                        'role': 'roles/storage.admin',
                        'members': ['group:' + owners_group,],
                    },
                    {
                        'role': 'roles/storage.objectCreator',
                        'members': ['group:cloud-storage-analytics@google.com'],
                    },
                    {
                        'role': 'roles/storage.objectViewer',
                        'members': ['group:' + auditors_group,],
                    },
                ],
            },
        },
    })

  # BigQuery dataset to hold audit logs.
  logs_dataset = context.properties.get('logs_bigquery_dataset')
  if logs_dataset:
    dataset_id = logs_dataset['name']
    resources.append({
        'name': dataset_id,
        'type': 'bigquery.v2.dataset',
        'properties': {
            'datasetReference': {
                'datasetId': dataset_id,
            },
            'location': logs_dataset['location'],
        },
    })

    # Update permissions for the dataset. This also removes the deployment
    # manager service account's access.
    resources.append({
        'name': 'update-' + dataset_id,
        'action': 'gcp-types/bigquery-v2:bigquery.datasets.patch',
        'properties': {
            'projectId':
                project_id,
            'datasetId':
                dataset_id,
            'access': [
                {
                    'role': 'OWNER',
                    'groupByEmail': owners_group,
                },
                {
                    'role': 'READER',
                    'groupByEmail': auditors_group,
                },
                {
                    'role': 'WRITER',
                    'userByEmail': logs_dataset['log_sink_service_account'],
                },
            ],
        },
        'metadata': {
            'dependsOn': [dataset_id],
        },
    })

  return {'resources': resources}
