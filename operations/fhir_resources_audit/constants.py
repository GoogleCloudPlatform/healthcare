"""Define all configs here"""


##################### REQUIRED CONFIG ###################################
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

##################### DO NOT MODIFY ###################################

# Base healthcare API url for FHIR stores
BASE_CHC_API_URL = "https://healthcare.googleapis.com/v1"


