# DataFlow Streaming Pipeline to read from HCAPI and write to Google Cloud Pub/Sub and Google Cloud Storage

The goal of this DataFlow streaming pipeline (Classic standalone deployment) is to consume FHIR notification from Google Cloud Healthcare API via Google Pub/Sub, transform FHIR resources as per Business requirements and send to downstream applications via Google Cloud Storage and Google Cloud Pub/Sub.
This solution is built using Google Cloud tools and services, such as Google Cloud Dataflow, Google Cloud Pub/Sub, Google Cloud Healthcare API and Google Cloud Storage. 
This pipeline will help users accelerate deploying streaming data pipelines from Google Cloud Healthcare API via Pub/Sub to Google Cloud Pub/Sub and Google Cloud Storage for downstream applications.