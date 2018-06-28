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

# This script downloads BigQuery queries from the specified project. The script
# prints out the queries to the console.

set -u -e

print_usage() {
  echo "Usage:"
  echo "  download_bq_queries.sh --project_id <PROJECT_ID>"
}

PROJECT_ID=""

while (( "$#" )); do
  if [[ $2 == --* ]]; then
    echo "Value of $1 starts with '--'. Missing value?"
    exit 1
  fi
  if [[ $1 == "--project_id" ]]; then
    PROJECT_ID=$2
  else
    echo "Unknown flag ${1}"
    exit 1
  fi
  shift 2
done

if [[ -z ${PROJECT_ID} ]]; then
  print_usage
  exit 1
fi

JOB_IDs=`bq ls -j -a --project_id=${PROJECT_ID} | grep query | awk '{print $1}'`
for job_id in ${JOB_IDs[*]}; do
  bq show --format=prettyjson --project_id=${PROJECT_ID} -j ${job_id} | \
    jq ".configuration.query.query"
done
