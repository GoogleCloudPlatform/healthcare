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
    zone = cluster['zone']
    machine_type = cluster['machine_type']
    # Create a new VM.
    initscripts = []
    for script in cluster['init_scripts']: 
        initscripts.append({
            'executableFile': script  
        })
    tags = []
    tags = cluster['tags']
    imageVersion = cluster['imageVersion']
    cluster_resource = {
        'name': name,
        'type': 'dataproc.v1.cluster',
        'properties': {
            'region': region,
            'clusterName': name,
            'config': {
                'gceClusterConfig': {
                   'zoneUri': zone,
                   'tags': tags,
                },
                'masterConfig': {
                   'numInstances': masternum,
                   'machineTypeUri': machine_type,
                },
                'workerConfig': {
                   'numInstances': workernum,
                   'machineTypeUri': machine_type,
                },
                'softwareConfig': {
                   'imageVersion': imageVersion,
                },
                'initializationActions': initscripts
            },
        },
       }

    resources.append(cluster_resource)

  return {'resources': resources}
