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
"""Setup file for Healthcare Imaging ML Toolkit."""

from setuptools import find_packages
from setuptools import setup

REQUIRED_PACKAGES = [
    'attrs',
    'google-auth',
    'retrying',
    # Typing is not needed for Python versions 3.5+
    'typing',
    'httplib2',
    'requests-toolbelt',
    'six',
    'urllib3',
    'google-auth-httplib2',
    'google-api-python-client',
    'tensorflow',
    'google-api-core',
    'google-cloud-pubsub',
    'absl-py',
    'numpy',
    # Mock is not needed for Python versions 3.3+
    'mock',
    'grpcio',
    'tensorflow-serving-api',
]

setup(
    name='hcls_imaging_ml_toolkit',
    version='0.1',
    install_requires=REQUIRED_PACKAGES,
    python_requires='>=3',
    packages=find_packages(),
    author='Google',
    author_email='noreply@google.com',
    license='Apache 2.0',
    url='https://github.com/GoogleCloudPlatform/healthcare',
    description='Toolkit for deploying ML models on GCP leveraging Google Cloud Healthcare API.'
)
