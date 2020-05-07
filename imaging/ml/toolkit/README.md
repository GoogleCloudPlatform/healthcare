# Imaging ML Toolkit

This directory contains useful helpers for deploying an ML Model using the Cloud Healthcare API in Python. Follow these instructions to get started.


# Installation

1. PIP install from github:

    ```
    pip3 install git+https://github.com/GoogleCloudPlatform/healthcare.git#subdirectory=imaging/ml/toolkit
    ```


# Example usage.

```
from hcls_imaging_ml_toolkit import tags

print(tags.SOP_CLASS_UID.number)
```

# Testing

## Unit tests
You can execute unit tests using:

```
python3 -m  unittest discover -v -p *_test.py
```
## Integration tests

Integration tests will be run automatically for all changes. They run on Cloud Build.

If you want to invoke the integration tests manually, run the following command:

```
cd healthcare/imaging/ml/toolkit/
gcloud builds submit --config cloudbuild.yaml .
```
