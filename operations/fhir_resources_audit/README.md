# FHIR Resources Audit

In accordance with FHIR specifications, provenance data for data lineage is stored using three different FHIR resources: Provenance, DocumentReference, and Device. Currently, the steps to obtain information about reconciled resources in final-fhir-store are tedious and only allows the user to track a single record at a time from ingestion to reconciliation. There is no way to query multiple records for data lineage at the same time.

This utility will take in minimal user input about the environment and output a detailed, easy to interpret JSON file with information about how source data gets ingested, transformed and reconciled into final FHIR resources.


## Before this Utility:

##### For FHIR and CDA batch pipelines:

1. Query operational-fhir-store to fetch DocumentReference ID  of source data file from GCS bucket.
2. Use this fetched DocumentReference ID as a reference to query provenance resource to get the details about transformed resources in the intermediate-fhir-store
3. To get the information about which pipeline/agent transformed source data to intermediate-fhir-store, you will get Device ID in the Provenance resource information which can be used to query Device resource and dataflow pipeline details.
4. To get information about reconciled resources in the final-fhir-store, you need to query the Provenance resource using intermediate-fhir-store resource ID.

##### For CSV batch pipelines:

1. Query operational-fhir-store to fetch DocumentReference ID  of source CSV data file from GCS bucket.
2. Since for CSV data harmonization, data first gets copied to Bigquery tables and from there it gets transformed into FHIR resources, get the details of DocumentReference ID  of Bigquery table using fetched DocumentReference ID  of source CSV data file.
3. Performed the same steps as for FHIR and CDA batch pipelines using DocumentReference ID of the Bigquery table.
4. Use this fetched DocumentReference ID of BQ table as a reference to query provenance resource to get the details about transformed resources in the intermediate-fhir-store.
5. To get the information about which pipeline/agent transformed source data to intermediate-fhir-store, you will get Device ID in the Provenance resource information which can be used to query Device resource and dataflow pipeline details.
6. To get information about reconciled resources in the final-fhir-store, you need to query the Provenance resource using intermediate-fhir-store resource ID.



#### Additionally, users must be familiar with the relationships between FHIR resources like Provenance, DocumentReference, and Device to get the required details.


## After this Utility:

With this utility the user needs to configure very minimal basic information in the config file and execute the main.py as shown below.


#### To run this utility program, configure values in the constants.py file 

```
# Source file GCS Path e.g. "gs://pgt-csv-input/ehr2/batch_02/csvs/patients.csv"
GCS_FILE_PATH = ""

# HDE Prefix e.g. "pgt"
HDE_PREFIX = ""           

# HDE env where data transformed e.g. "stage", "synth", "dev", "prod"
HDE_ENV = ""

# HDE FHIR store location e.g. "us", "us-central1", "us-east4"
FHIR_STORE_LOC = ""

# Need reconciliation information Boolean value - default: False e.g. True
RECON_INFO = False

# Output BQ table in format "{PROJECT_ID}:{DATASET_ID}.{TABLE_NAME}" e.g. "hde14-stage-data:ehr2_batch_01_9f77ec70.provenance_tbl"
OUTPUT_BQ_TBL = ""

# Local file path for storing the output e.g. "output.json", "/home/user/output.json"
OUTPUT_FILE_PATH = ""

```

#### And Execute main.py file. 
```
python main.py
```


### That’s it!

The utility will provide a summary of the information in the logs.

![Log output image](./output_images/log_op.png)

A detailed information about the generated FHIR resources can be written to a BigQuery table and/or JSON document in the user-configured output BigQuery table and output file path. This is an optional step.

![json output image](./output_images/json_op.png)

![Bigquery table output image](./output_images/bqtable_op.png)
