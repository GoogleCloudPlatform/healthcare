# FHIR Resources Audit

In accordance with FHIR specifications, provenance data for data lineage is stored using three different FHIR resources, Provenance, DocumentReference, and Device.

## Relationship between these FHIR Resources:
 * Provenance and DocumentReference: The Provenance resource can reference a DocumentReference as the source entity of an activity (e.g., creation, modification). This helps track the origin and evolution of the FHIR resource.

 * Provenance and Device: Similarly, Provenance can reference a Device used in an activity. This provides insights into the device's role in generating FHIR resources.

 * DocumentReference and Device: A DocumentReference can contain information about the devices used in its creation. This could include device settings, data pipeline information.


To track FHIR resources, you need to query three different resources using the relationships listed above.

This utility will help you track all FHIR resources. It will create a JSON output that shows how data was transformed into the intermediate-fhir-store and reconciled into the final-fhir-store.



#### To run this utility program, you need to add your configuration values in the constants.py file 
```
# Source file format e.g. "csv", "ndjson", "xml"
FILE_FORMAT = ""

# Source file GCS Path e.g. "gs://pgt-csv-input/ehr2/batch_02/csvs/patients.csv"
GCS_FILE_PATH = ""

# HDE Prefix e.g. "pgt"
HDE_PREFIX = ""           

# HDE env where data transformed e.g. "stage", "synth", "dev", "prod"
HDE_ENV = ""

# HDE FHIR store location e.g. "us", "us-central1", "us-east4"
FHIR_STORE_LOC = ""


# Local file path for storing the output e.g. "output.json", "/home/user/output.json"
OUTPUT_FILE_PATH = ""

```


#### And Execute main.py file. 
```
python main.py

```


### That’s it!

The utility will generate a JSON document in the user-configured output file path and provide information about the generated FHIR resources.