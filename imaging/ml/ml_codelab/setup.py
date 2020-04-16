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
"""Module used to setup Dataflow depenedencies for data preprocessing."""

from setuptools import find_packages
from setuptools import setup
from setuptools.command.install import install
from setuptools.dist import Distribution


class _InstallPlatlib(install):

  def finalize_options(self):
    install.finalize_options(self)
    self.install_lib = self.install_platlib


class _BinaryDistribution(Distribution):
  """This class is needed in order to create OS specific wheels."""

  def is_pure(self):
    return False

  def has_ext_modules(self):
    return True


REQUIRED_PACKAGES = [
    'absl-py>=0.7,<0.9', 'apache-beam[gcp]>=2.16,<=2.18', 'numpy>=1.16,<2',
    'protobuf>=3.7,<4', 'psutil>=5.6,<6', 'six>=1.12,<2',
    'tensorflow_hub>=0.7.0', 'tensorflow==1.15.*'
]

setup(
    name='breast_density_model',
    version='0.1',
    install_requires=REQUIRED_PACKAGES,
    packages=find_packages(),
    include_package_data=True,
    namespace_packages=[],
    distclass=_BinaryDistribution,
    package_data={'': ['*.lib', '*.pyd', '*.so']},
    python_requires='>=3.5',
    author='Google',
    author_email='noreply@google.com',
    license='Apache 2.0',
    url='https://github.com/GoogleCloudPlatform/healthcare',
    cmdclass={'install': _InstallPlatlib},
    description='Breast Density Classification Model.')
