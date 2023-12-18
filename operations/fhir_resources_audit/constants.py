"""Define all configs here"""


##################### REQUIRED CONFIG ###################################

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

##################### DO NOT MODIFY ###################################

# Base healthcare API url for FHIR stores
BASE_CHC_API_URL = "https://healthcare.googleapis.com/v1"


