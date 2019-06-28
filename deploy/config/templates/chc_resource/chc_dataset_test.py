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
"""Tests for healthcare.deploy.config.templates.chc_resource.

These tests check that the template is free from syntax errors and generates
the expected resources.
"""

from absl.testing import absltest

from deploy.config.templates.chc_resource import chc_dataset


class TestCHCDataset(absltest.TestCase):

  def test_chc_dataset(self):

    class FakeContext(object):
      env = {
          'project': 'my-project',
      }
      properties = {
          'datasetId': 'test_chc_dataset',
          'location': 'us-central1',
          'dicomStores': [{
              'dicomStoreId': 'test_chc_dicom_store'
          }],
          'fhirStores': [{
              'fhirStoreId': 'test_chc_fhir_store'
          }],
          'hl7V2Stores': [{
              'hl7V2StoreId': 'test_chc_hl7v2_store'
          }],
      }

    generated = chc_dataset.generate_config(FakeContext())

    expected = {
        'resources': [{
            'name': 'test_chc_dataset',
            'type': 'my-project/cloud-healthcare:projects.locations.datasets',
            'properties': {
                'parent': 'projects/my-project/locations/us-central1',
                'datasetId': 'test_chc_dataset'
            }
        }, {
            'name':
                'test_chc_dataset_test_chc_dicom_store',
            'type':
                'my-project/cloud-healthcare:projects.locations.datasets.dicomStores',
            'properties': {
                'parent': '$(ref.test_chc_dataset.name)',
                'dicomStoreId': 'test_chc_dicom_store'
            }
        }, {
            'name':
                'test_chc_dataset_test_chc_fhir_store',
            'type':
                'my-project/cloud-healthcare:projects.locations.datasets.fhirStores',
            'properties': {
                'parent': '$(ref.test_chc_dataset.name)',
                'fhirStoreId': 'test_chc_fhir_store'
            }
        }, {
            'name':
                'test_chc_dataset_test_chc_hl7v2_store',
            'type':
                'my-project/cloud-healthcare:projects.locations.datasets.hl7V2Stores',
            'properties': {
                'parent': '$(ref.test_chc_dataset.name)',
                'hl7V2StoreId': 'test_chc_hl7v2_store'
            }
        }]
    }

    index = 0
    self.assertLen(generated['resources'], len(expected['resources']))
    while index < len(generated['resources']):
      self.assertEqual(generated['resources'][index],
                       expected['resources'][index])
      index += 1


if __name__ == '__main__':
  absltest.main()
