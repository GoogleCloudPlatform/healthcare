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

"""Tests for healthcare.deploy.templates.gce_vms.

These tests check that the template is free from syntax errors and generates
the expected resources.

To run tests, run `python -m unittest tests.gce_vms` from the templates
directory.
"""

import unittest

from templates import gce_vms


class TestGceVmsTemplate(unittest.TestCase):

  def test_template_expansion(self):
    class FakeContext(object):
      env = {
          'deployment': 'my-deployment',
          'project': 'my-project',
      }
      startup_script_str='echo "abc"\necho "def"\n'
      properties = {
          'gce_instances': [
              {
                  'name': 'work-machine-1',
                  'zone': 'asia-southeast1-a',
                  'machine_type': 'n1-standard-1',
                  'boot_image_name': 'global/images/my-boot-disk',
                  'start_vm': False,
              }, {
                  'name': 'work-machine-2',
                  'zone': 'asia-southeast1-b',
                  'machine_type': 'n1-standard-1',
                  'boot_image_name': (
                      'projects/debian-cloud/global/images/family/debian-9'),
                  'start_vm': True,
                  'metadata': {
                    'items':[{
                        'key': 'startup-script',
                        'value': startup_script_str
                    }]
                  }
              },
          ],
          'firewall_rules': [
              {
                  'name': 'firewall-allow-rstudio',
                  'allowed': [{
                      'IPProtocol': 'tcp',
                      'ports': ['8787'],
                  }],
                  'sourceRanges': ['0.0.0.0/0'],
              },
          ],
      }

    generated = gce_vms.generate_config(FakeContext())

    expected = {
        'resources': [{
            'name': 'work-machine-1',
            'type': 'compute.v1.instance',
            'properties': {
                'zone': 'asia-southeast1-a',
                'machineType': (
                    'zones/asia-southeast1-a/machineTypes/n1-standard-1'),
                'disks': [{
                    'deviceName': 'boot',
                    'type': 'PERSISTENT',
                    'boot': True,
                    'autoDelete': True,
                    'initializeParams': {
                        'sourceImage': 'global/images/my-boot-disk',
                    },
                }],
                'networkInterfaces': [{
                    'network': 'global/networks/default',
                    'accessConfigs': [{
                        'name': 'External NAT',
                        'type': 'ONE_TO_ONE_NAT',
                    }],
                }],
                'metadata': {
                },
            }
        }, {
            'name': 'stop-work-machine-1',
            'action': 'gcp-types/compute-v1:compute.instances.stop',
            'properties': {
                'instance': 'work-machine-1',
                'zone': 'asia-southeast1-a',
            },
            'metadata': {
                'dependsOn': ['work-machine-1'],
                'runtimePolicy': ['CREATE'],
            },
        }, {
            'name': 'work-machine-2',
            'type': 'compute.v1.instance',
            'properties': {
                'zone': 'asia-southeast1-b',
                'machineType': (
                    'zones/asia-southeast1-b/machineTypes/n1-standard-1'),
                'disks': [{
                    'deviceName': 'boot',
                    'type': 'PERSISTENT',
                    'boot': True,
                    'autoDelete': True,
                    'initializeParams': {
                        'sourceImage': ('projects/debian-cloud/global/'
                                        'images/family/debian-9'),
                    },
                }],
                'networkInterfaces': [{
                    'network': 'global/networks/default',
                    'accessConfigs': [{
                        'name': 'External NAT',
                        'type': 'ONE_TO_ONE_NAT',
                    }],
                }],
                'metadata': {
                    'items': [{'value': 'echo "abc"\necho "def"\n',
                    'key': 'startup-script'}]
                },
            }
        }, {
            'name': 'firewall-allow-rstudio',
            'type': 'compute.v1.firewall',
            'properties': {
                'allowed': [{
                    'IPProtocol': 'tcp',
                    'ports': ['8787'],
                }],
                'sourceRanges': ['0.0.0.0/0'],
            },
        }]
    }

    self.assertEqual(generated, expected)


if __name__ == '__main__':
  unittest.main()
