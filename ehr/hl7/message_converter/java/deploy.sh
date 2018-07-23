#!/bin/bash

set -e

API_ADDR_PREFIX="https://www.example.com/v1"
PROJECT_ID="project-id"
LOCATION_ID="location-id"
DATASET_ID="dataset-id"
FHIR_STORE_ID="fhir-store-id"
PUBSUB_PROJECT_ID="pubsub-project-id"
PUBSUB_SUBSCRIPTION_ID="pubsub-subscription-id"

echo "Building docker image..."
gradle dockerBuildImage

echo "Uploading to GCR..."
docker tag hl7v2_to_fhir_converter gcr.io/${PROJECT_ID}/hl7v2-to-fhir-converter:latest
docker push gcr.io/${PROJECT_ID}/hl7v2-to-fhir-converter:latest

cat << EOF > converter.yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: hl7v2-to-fhir-converter-deployment
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: hl7v2-to-fhir-converter
    spec:
      containers:
        - name: hl7v2-to-fhir-converter
          image: gcr.io/${PROJECT_ID}/hl7v2-to-fhir-converter:latest
          command:
            - "/healthcare/bin/healthcare"
            - "--apiAddrPrefix=${API_ADDR_PREFIX}"
            - "--fhirProjectId=${PROJECT_ID}"
            - "--fhirLocationId=${LOCATION_ID}"
            - "--fhirDatasetId=${DATASET_ID}"
            - "--fhirStoreId=${FHIR_STORE_ID}"
            - "--pubsubProjectId=${PUBSUB_PROJECT_ID}"
            - "--pubsubSubscription=${PUBSUB_SUBSCRIPTION_ID}"
EOF

echo "Creating GKE cluster..."
gcloud container clusters create hl7v2-to-fhir-converter --scopes https://www.googleapis.com/auth/pubsub,https://www.googleapis.com/auth/cloud-healthcare

echo "Creating GKE deployment..."
kubectl create -f converter.yaml
