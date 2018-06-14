#!/bin/bash

# Copyright 2018 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     https://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script detects BigQuery schemas for csv data files.
# Data files are expected to be compressed csv files in *.csv.gz format.

set -u -e

print_usage() {
  echo "Usage:"
  echo "  generate_schemas.sh --input_dir <INPUT_DIR> --schema_output_dir <SCHEMA_OUTPUT_DIR>"
}

INPUT_DIR=""
SCHEMA_OUTPUT_DIR=""

while (( "$#" )); do
  if [[ $1 == "--input_dir" ]]; then
    INPUT_DIR=$2
  elif [[ $1 == "--schema_output_dir" ]]; then
    SCHEMA_OUTPUT_DIR=$2
  else
    echo "Unknown flag ${1}."
    exit 1
  fi
  shift 2
done

if [[ -z ${INPUT_DIR} ]] || [[ -z ${SCHEMA_OUTPUT_DIR} ]]; then
  print_usage
  exit 1
fi

GOEXEC=`which go`
BQ_SCHEMA_DETECTOR_PATH=`dirname "$0"`/bq_schema_detector.go

# Scan all file of *.csv.gz format in the given directory.
for file in $INPUT_DIR/*.csv.gz; do
  echo "Processing ${file}."
  # Decompress the file.
  gzip -d -k ${file}
  csv_file=${file:0:(-3)}
  # Generate schema file for csv file. Output file will have "csv.gz.schema"
  # extension.
  schema_file=${SCHEMA_OUTPUT_DIR}/$(basename ${file}).schema
  ${GOEXEC} run ${BQ_SCHEMA_DETECTOR_PATH} -input_csv=${csv_file} \
    -output_file=${schema_file}
  # Remove the decompressed file.
  rm ${csv_file}
done
