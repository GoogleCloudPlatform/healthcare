"""Define all configs here"""


##################### REQUIRED CONFIG ###################################

# Source file GCS Path e.g. "gs://pgt-csv-input/ehr2/batch_02/csvs/patients.csv"
GCS_FILE_PATH = "gs://hde-accelerators-test-fhir-data/v1.4.0-rc2/MessageHeader.ndjson"

# HDE Prefix e.g. "pgt"
HDE_PREFIX = "hde140-uat-12"           

# HDE env where data transformed e.g. "stage", "synth", "dev", "prod"
HDE_ENV = "synth"

# HDE FHIR store location e.g. "us", "us-central1", "us-east4"
FHIR_STORE_LOC = "us"

# Need reconciliation information Boolean value e.g. True, False
RECON_INFO = True

# Output BQ table in format "{PROJECT_ID}:{DATASET_ID}.{TABLE_NAME}" e.g. "hde140-uat-12-stage-data:ehr2_batch_01_9f77ec70.provenance_tbl"
OUTPUT_BQ_TBL = "hde140-uat-12-synth-data.ehr1_batch_01_57913d37.my_table_fhir"

# Local file path for storing the output e.g. "output.json", "/home/user/output.json"
OUTPUT_FILE_PATH = "fhir_final_output.json"

##################### DO NOT MODIFY ###################################

# Base healthcare API url for FHIR stores
BASE_CHC_API_URL = "https://healthcare.googleapis.com/v1"


