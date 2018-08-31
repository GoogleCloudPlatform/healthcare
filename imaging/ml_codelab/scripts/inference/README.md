# Inference Module

This directory contains the inference module for the breast density model. The
inference module is designed to be packaged into a Docker container and deployed
to GKE.

## Packing into Docker container and pushing to Google Container Registry

```shell
gcloud builds submit --config cloudbuild.yaml --timeout 1h .
```

## Deploying to GKE

```shell
PROJECT_ID=..
SUBSCRIPTION_NAME=..
MODEL_PATH=..

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
            - "--project_id=${PROJECT_ID}"
            - "--subscription_name=${SUBSCRIPTION_NAME}"
            - "--model_path=${MODEL_PATH}"
EOF
```
