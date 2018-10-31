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

"""Creates new dataproc cluster with specified zone,region, number of worker, master nodes, machine_type"""


def generate_config(context):
  """Generate Deployment Manager configuration."""
  resources = []

  for cluster  in context.properties['dataproc_clusters']:
    name = cluster['name']
    region  = cluster['region']
    workernum  = cluster['workernum']
    masternum  = cluster['masternum']
#    machine_type = 'zones/{}/machineTypes/{}'.format(cluster['zone'], cluster['machine_type'])
#    zone = 'zones/{}'.format(cluster['zone'])
    zone = cluster['zone']
    machine_type = cluster['machine_type']
    # Create a new VM.
    cluster_resource = {
        'name': name,
        'type': 'dataproc.v1.cluster',
        'properties': {
            'region': region,
            'clusterName': name,
            'config': {
                'gceClusterConfig': {
                   'zoneUri': zone,
                },
                'masterConfig': {
                   'numInstances': masternum,
                   'machineTypeUri': machine_type,
                },
                'workerConfig': {
                   'numInstances': workernum,
                   'machineTypeUri': machine_type,
               },
            },
        },
    }

    resources.append(cluster_resource)

  return {'resources': resources}
