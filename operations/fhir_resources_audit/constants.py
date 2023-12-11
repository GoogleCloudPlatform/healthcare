"""Define all configs here"""


##################### REQUIRED CONFIG ###################################
# Source file format e.g. "csv", ".ndjson", "xml"
FILE_FORMAT = "csv"

# Source file GCS Path e.g. "gs://pgt-csv-input/ehr2/batch_02/csvs/patients.csv"
GCS_FILE_PATH = "gs://hde140-uat-12-stage-csv-input/ehr2/batch_02/csvs/patients.csv"

# HDE Prefix e.g. "pgt"
HDE_PREFIX = "hde140-uat-12"           

# HDE env where data transformed e.g. "stage", "synth", "dev", "prod"
HDE_ENV = "stage"

# HDE FHIR store location e.g. "us", "us-central1", "us-east4"
FHIR_STORE_LOC = "us"
