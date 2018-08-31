# Copyright 2018 Google LLC. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
from setuptools import find_packages
from setuptools import setup

REQUIRED_PACKAGES = [
    'pydicom',
    'requests-toolbelt',
    'google-api-python-client',
    'google-api-core',
    # Pin googleapis-common-proto until issue is resolved.
    # https://github.com/GoogleCloudPlatform/google-cloud-python/issues/5703
    'googleapis-common-protos==1.5.3',
    'google-cloud-pubsub',
    'httplib2',
    'oauth2client',
    'google-cloud-automl'
]

setup(
    name='breast_density_inference_module',
    version='0.1',
    install_requires=REQUIRED_PACKAGES,
    packages=find_packages(),
    include_package_data=True,
    description='Breast Density Classification Inference Module.')
