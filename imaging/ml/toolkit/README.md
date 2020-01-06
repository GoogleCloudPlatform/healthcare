# Imaging ML Toolkit

This directory contains useful helpers for deploying an ML Model using the Cloud Healthcare API in Python. Follow these instructions to get started.


# Installation
Clone the library:
1. git clone https://github.com/GoogleCloudPlatform/healthcare.git
Install dependencies:
2. cd healthcare
   pip install -r ./imaging/ml/toolkit/requirements.txt


# Example usage.

```
from toolkit import tags

print(tags.SOP_CLASS_UID.number)
```
