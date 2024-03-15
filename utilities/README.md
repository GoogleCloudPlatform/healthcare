# Healthcare Toolkit

The healthcare toolkit is designed to guide various data movement patterns including data ingestion, consumption, and auditing. The utilities provide data movement patterns between various sources/sinks and Google Cloud healthcare products. 
Healthcare Toolkit Utilities.

## Ingest
1. [Kafka to FHIR Store](https://github.com/GoogleCloudPlatform/healthcare/tree/master/utilities/ingest/dataflow_kafka_to_fhirstore) - Consume messages via Kafka, and write to a GCP FHIR Store.
2. [Pub/Sub to FHIR Store](https://github.com/GoogleCloudPlatform/healthcare/tree/master/utilities/ingest/dataflow_pubsub_to_fhirstore) - Consume messages via Pub/Sub, and write to a GCP FHIR Store.

## Consume
1. [FHIR Store to GCS](https://github.com/GoogleCloudPlatform/healthcare/tree/master/utilities/consume/dataflow_fhirstore_bq_consume) - Consumes a notification from a GCP FHIR Store, and writes the message to GCS.
2. [FHIR Store to Pub/Sub + GCS](https://github.com/GoogleCloudPlatform/healthcare/tree/master/utilities/consume/dataflow_fhirstore_to_gcs_pubsub) - Consumes a notification from a GCP FHIR Store, and writes it to GCP Pub/Sub and Google Cloud Storage.


## Audit

1. [FHIR Resource Auditing](https://github.com/GoogleCloudPlatform/healthcare/tree/master/utilities/audit/fhir_resources_audit) - Provides a JSON with information about how source data gets ingested, transformed and reconciled into final FHIR resources.


## Disclaimer
This repository and its contents are not an official Google Product.

##Contact
Share your feedback, ideas, thoughts on [Github Issues](https://github.com/GoogleCloudPlatform/healthcare/issues). If you’d like to start a discussion, please do so via [Github Discussions](https://github.com/GoogleCloudPlatform/healthcare/discussions). Please contact [healthcare-toolkit-support@googlegroups.com](healthcare-toolkit-support@googlegroups.com), should you have any use-cases or blockers with these utilities.
