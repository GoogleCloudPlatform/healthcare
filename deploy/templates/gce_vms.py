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

"""Creates new GCE VMs with specified zone, machine type and boot image."""


def generate_config(context):
  """Generate Deployment Manager configuration."""
  resources = []

  for vm in context.properties['gce_instances']:
    vm_name = vm['name']
    zone = vm['zone']
    machine_type = 'zones/{}/machineTypes/{}'.format(zone, vm['machine_type'])
    boot_image = vm['boot_image_name']
    metadata = vm.get('machine_type', {})

    # Create a new VM.
    resources.append({
        'name': vm_name,
        'type': 'compute.v1.instance',
        'properties': {
            'zone': zone,
            'machineType': machine_type,
            'disks': [{
                'deviceName': 'boot',
                'type': 'PERSISTENT',
                'boot': True,
                'autoDelete': True,
                'initializeParams': {
                    'sourceImage': boot_image,
                },
            }],
            'networkInterfaces': [{
                'network': 'global/networks/default',
                'accessConfigs': [{
                    'name': 'External NAT',
                    'type': 'ONE_TO_ONE_NAT',
                }],
            }],
            'metadata': metadata
        },
    })

    # After the VM is created, shut it down (if start_vm is False).
    if not vm['start_vm']:
      resources.append({
          'name': 'stop-' + vm_name,
          'action': 'gcp-types/compute-v1:compute.instances.stop',
          'properties': {
              'instance': vm_name,
              'zone': zone,
          },
          'metadata': {
              'dependsOn': [vm_name],
              'runtimePolicy': ['CREATE'],
          },
      })

  # Create firewall rules (if any).
  for rule in context.properties.get('firewall_rules'):
    name = rule.pop('name')
    resources.append({
        'name': name,
        'type': 'compute.v1.firewall',
        'properties': rule
    })

  return {'resources': resources}
