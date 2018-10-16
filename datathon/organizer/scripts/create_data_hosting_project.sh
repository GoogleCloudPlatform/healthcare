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

# This script sets up a data hosting project for the datathon event. Uploading
# data function will be included in separate scripts.

set -u -e

print_usage() {
  echo "Usage:"
  echo "  create_data_hosting_project.sh --owners_group <OWNERS_GROUP> \ "
  echo "      --editors_group <EDITORS_GROUP> \ "
  echo "      --data_hosting_project_id <DATA_HOSTING_PROJECT_ID> \ "
  echo "      --billing_account <BILLING_ACCOUNT> \ "
  echo "      --audit_project_id <AUDIT_PROJECT_ID> \ "
  echo "      --audit_dataset_id <AUDIT_DATASET_ID>"
}

OWNERS_GROUP=""
EDITORS_GROUP=""
DATA_HOSTING_PROJECT_ID=""
BILLING_ACCOUNT=""
AUDIT_PROJECT_ID=""
AUDIT_DATASET_ID=""

while (( "$#" )); do
  if [[ $2 == --* ]]; then
    echo "Value of $1 starts with '--'. Missing value?"
    exit 1
  fi
  if [[ $1 == "--owners_group" ]]; then
    OWNERS_GROUP=$2
  elif [[ $1 == "--editors_group" ]]; then
    EDITORS_GROUP=$2
  elif [[ $1 == "--data_hosting_project_id" ]]; then
    DATA_HOSTING_PROJECT_ID=$2
  elif [[ $1 == "--billing_account" ]]; then
    BILLING_ACCOUNT=$2
  elif [[ $1 == "--audit_project_id" ]]; then
    AUDIT_PROJECT_ID=$2
  elif [[ $1 == "--audit_dataset_id" ]]; then
    AUDIT_DATASET_ID=$2
  else
    echo "Unknown flag ${1}"
    exit 1
  fi
  shift 2
done

if [[ -z ${OWNERS_GROUP} ]] || [[ -z ${EDITORS_GROUP} ]] || \
     [[ -z ${DATA_HOSTING_PROJECT_ID} ]] || [[ -z ${BILLING_ACCOUNT} ]] || \
     [[ -z ${AUDIT_PROJECT_ID} ]] || [[ -z ${AUDIT_DATASET_ID} ]]; then
  print_usage
  exit 1
fi
STATE_FILE="$0".state
# A list of state checkpoints for the script to resume to.
STATE_SET_PERMISSION="SET_PERMISSION"
STATE_SET_BILLING="SET_BILLING"
STATE_ENABLE_LOGGING="ENABLE_LOGGING"

if [[ ! -e ${STATE_FILE} ]]; then
  echo "Creating Google Cloud project '${DATA_HOSTING_PROJECT_ID}' to host " \
    "data."
  gcloud projects create "${DATA_HOSTING_PROJECT_ID}"
  PROJECT_NUMBER=$(gcloud projects describe ${DATA_HOSTING_PROJECT_ID} \
    --format='value(projectNumber)')
  echo "Data hosting project created: ID=${DATA_HOSTING_PROJECT_ID}," \
    "Number=${PROJECT_NUMBER}."
  echo ${STATE_SET_PERMISSION} > ${STATE_FILE}
else
  echo "Skip creating project since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_SET_PERMISSION} ]]; then
  PARENT_TYPE=$(gcloud projects describe ${DATA_HOSTING_PROJECT_ID} \
    --format='value(parent.type)')
  if [[ ${PARENT_TYPE} == "organization" ]]; then
    echo "Setting ${OWNERS_GROUP} as owner for project" \
      "'${DATA_HOSTING_PROJECT_ID}'."
    gcloud projects add-iam-policy-binding ${DATA_HOSTING_PROJECT_ID} \
      --member="group:${OWNERS_GROUP}" --role="roles/owner"
    echo "Setting ${EDITORS_GROUP} as editor for project" \
      "'${DATA_HOSTING_PROJECT_ID}'."
    gcloud projects add-iam-policy-binding ${DATA_HOSTING_PROJECT_ID} \
      --member="group:${EDITORS_GROUP}" --role="roles/editor"
    echo "Revoking individual owner access."
    USER_EMAIL=$(gcloud config list account --format "value(core.account)")
    gcloud projects remove-iam-policy-binding ${DATA_HOSTING_PROJECT_ID} \
      --member="user:${USER_EMAIL}" --role="roles/owner"
  else
    echo "Because the project is created without an organization, it is not"
    echo "possible to use a group as an owner of the project. Granting"
    echo "${OWNERS_GROUP} permission to change IAM configuration, so that"
    echo "members of this group can add individual owners (including"
    echo "themselves) when needed."
    gcloud projects add-iam-policy-binding "${DATA_HOSTING_PROJECT_ID}" \
      --member="group:${OWNERS_GROUP}" \
      --role="roles/resourcemanager.projectIamAdmin"
  fi
  echo ${STATE_SET_BILLING} > ${STATE_FILE}
else
  echo "Skip setting permissions since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_SET_BILLING} ]]; then
  echo "Setting billing account ${BILLING_ACCOUNT} for project" \
    "'${DATA_HOSTING_PROJECT_ID}'."
  gcloud beta billing projects link --billing-account "${BILLING_ACCOUNT}" \
    "${DATA_HOSTING_PROJECT_ID}"
  echo ${STATE_ENABLE_LOGGING} > ${STATE_FILE}
else
  echo "Skip setting billing since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_ENABLE_LOGGING} ]]; then
  echo "Enabling audit logging for the project."
  SINK_NAME=audit-logs-to-bigquery
  AUDIT_DATASET_URL="bigquery.googleapis.com/projects/${AUDIT_PROJECT_ID}/datasets/${AUDIT_DATASET_ID}"
  if `gcloud logging sinks list | grep -q ${SINK_NAME}`; then
    gcloud --quiet logging sinks delete ${SINK_NAME}
  fi
  LOG_SERVICE_ACCT=$((gcloud --quiet logging sinks create "${SINK_NAME}" \
    "${AUDIT_DATASET_URL}" --project ${DATA_HOSTING_PROJECT_ID}) 2>&1 \
    | grep 'Please remember to grant' | \
    sed -e 's/^[^`]*`serviceAccount://' -e 's/`.*$//')
  # Add the log service account as WRITER to the audit BigQuery dataset.
  TEMP=`tempfile`
  bq show --format=prettyjson "${AUDIT_PROJECT_ID}:${AUDIT_DATASET_ID}" | jq \
    ".access+[{\"userByEmail\":\"${LOG_SERVICE_ACCT?}\",\"role\":\"WRITER\"}]" \
    | jq '{access:.}' > ${TEMP}
  bq update --source ${TEMP} "${AUDIT_PROJECT_ID}:${AUDIT_DATASET_ID}"
else
  echo "Skip configuring logging since it has previously finished."
fi

rm -f ${STATE_FILE}

echo "Data hosting project setup finished."
