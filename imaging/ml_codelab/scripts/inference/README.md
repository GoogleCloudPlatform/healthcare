# Inference Module

This directory contains the inference module for the breast density model. The
inference module is designed to be packaged into a Docker container and deployed
to GKE.

## Packing into Docker container and pushing to Google Container Registry

```shell
gcloud builds submit --config cloudbuild.yaml --timeout 1h .
```

## Deploying to GKE

Note that all paths are fully qualified (`projects/PROJECT-ID/...`)
```shell
PUBLISHER_TOPIC_PATH=..
SUBSCRIPTION_PATH=..
MODEL_PATH=..
DICOM_STORE_PATH=..

cat <<EOF | kubectl create -f -
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: inference-module
  namespace: default
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: inference-module
    spec:
      containers:
        - name: inference-module
          image: gcr.io/dicomweb-gke/inference-module:latest
          command:
            - "/opt/inference_module/bin/inference_module"
            - "--subscription_path=${SUBSCRIPTION_PATH}"
            - "--publisher_topic_path=${PUBLISHER_TOPIC_PATH}"
            - "--model_path=${MODEL_PATH}"
            - "--dicom_store_path=${DICOM_STORE_PATH}"
            - "--prediction_service=AutoML"
EOF
```
