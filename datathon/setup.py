"""Install datathon python packages."""

import setuptools

setuptools.setup(
    name='google-cloud-healthcare-datathon',
    version='0.1',
    install_requires=[
        'apache-beam[gcp]', 'google-cloud-storage', 'tf-nightly', 'typing',
        'numpy'
    ],
    packages=setuptools.find_packages())
