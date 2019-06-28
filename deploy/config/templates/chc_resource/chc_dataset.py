# Copyright 2019 Google LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""This template creates Healthcare dataset."""


def generate_config(context):
  """Entry point for the deployment resources."""

  properties = context.properties
  project_id = context.env['project']
  dataset_id = properties['datasetId']
  location = properties['location']
  supported_types = {
      'dicomStores': 'dicomStoreId',
      'fhirStores': 'fhirStoreId',
      'hl7V2Stores': 'hl7V2StoreId'
  }
  type_provider_name = properties.get('type_provider', 'cloud-healthcare')

  resources = []
  dataset_location = 'projects/' + project_id + '/locations/' + location
  resources.append({
      'name':
          dataset_id,
      'type':
          project_id + '/' + type_provider_name +
          ':projects.locations.datasets',
      'properties': {
          'parent': dataset_location,
          'datasetId': dataset_id
      }
  })
  for res_type, id_tag in supported_types.items():
    stores = properties.get(res_type, [])
    for store in stores:
      resources.append({
          'name':
              dataset_id + '_' + store[id_tag],
          'type':
              project_id + '/' + type_provider_name +
              ':projects.locations.datasets.' + res_type,
          'properties': {
              'parent': '$(ref.' + dataset_id + '.name)',
              id_tag: store[id_tag]
          }
      })

  return {'resources': resources}
