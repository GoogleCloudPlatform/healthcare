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

# This script sets up an audit project for the datathon event. This audit
# project will contain a BigQuery dataset to house all logs from the datathon
# relevant projects.
# This script must be run by a member of the specified owners group.
set -u -e

print_usage() {
  echo "Usage:"
  echo "  create_audit_project.sh --owners_group <OWNERS_GROUP> \ "
  echo "      --audit_project_id <AUDIT_PROJECT_ID> \ "
  echo "      --billing_account <BILLING_ACCOUNT> \ "
  echo "      --auditors_group <AUDITORS_GROUP>"
}

OWNERS_GROUP=""
AUDIT_PROJECT_ID=""
BILLING_ACCOUNT=""
AUDITORS_GROUP=""

while (( "$#" )); do
  if [[ $1 == "--owners_group" ]]; then
    OWNERS_GROUP=$2
  elif [[ $1 == "--audit_project_id" ]]; then
    AUDIT_PROJECT_ID=$2
  elif [[ $1 == "--billing_account" ]]; then
    BILLING_ACCOUNT=$2
  elif [[ $1 == "--auditors_group" ]]; then
    AUDITORS_GROUP=$2
  else
    echo "Unknown flag ${1}"
    exit 1
  fi
  shift 2
done

if [[ -z ${OWNERS_GROUP} ]] || [[ -z ${AUDIT_PROJECT_ID} ]] \
     || [[ -z ${BILLING_ACCOUNT} ]] || [[ -z ${AUDITORS_GROUP} ]]; then
  print_usage
  exit 1
fi

USER_EMAIL=$(gcloud config list account --format "value(core.account)")

echo "Creating Google Cloud project '${AUDIT_PROJECT_ID}' to audit access."
gcloud projects create "${AUDIT_PROJECT_ID}"
PROJECT_NUMBER=$(gcloud projects describe ${AUDIT_PROJECT_ID} \
                        --format='value(projectNumber)')
echo "Audit project created: ID=${AUDIT_PROJECT_ID}, Number=${PROJECT_NUMBER}."

PARENT_TYPE=$(gcloud projects describe ${AUDIT_PROJECT_ID} \
                     --format='value(parent.type)')
if [[ ${PARENT_TYPE} == "organization" ]]; then
  echo "Setting ${OWNERS_GROUP} as owner for project '${AUDIT_PROJECT_ID}'."
  gcloud projects add-iam-policy-binding ${AUDIT_PROJECT_ID} \
    --member="group:${OWNERS_GROUP}" --role="roles/owner"
  echo "Revoking individual owner access."
  gcloud projects remove-iam-policy-binding ${AUDIT_PROJECT_ID} \
    --member="user:${USER_EMAIL}" --role="roles/owner"
else
  echo "Skipping step for setting ${OWNERS_GROUP} as the owner of the project."
  echo "This is because we have not set up an organization for datathons."
fi

echo "Setting billing account ${BILLING_ACCOUNT} for project
  '${AUDIT_PROJECT_ID}'."
gcloud beta billing projects link --billing-account "${BILLING_ACCOUNT}" \
  "${AUDIT_PROJECT_ID}"

echo "Enabling the following Google Cloud Platform services"
echo "  - Deployment Manager"
echo "  - BigQuery"
gcloud services enable deploymentmanager bigquery --project ${AUDIT_PROJECT_ID}

echo "Creating BigQuery dataset for audit logs."
LOCATION=US
AUDIT_DATASET_ID=audit_logs
TEMP=`tempfile`
cat <<EOF >>${TEMP}
resources:
- name: big-query-dataset
  type: bigquery.v2.dataset
  properties:
    datasetReference:
      datasetId: "${AUDIT_DATASET_ID?}"
    access:
      - role: 'OWNER'
        groupByEmail: "${OWNERS_GROUP?}"
      - role: 'READER'
        groupByEmail: "${AUDITORS_GROUP?}"
    location: "${LOCATION?}"
EOF
gcloud deployment-manager deployments create create-audit-logs-ds \
  --config=${TEMP} --project ${AUDIT_PROJECT_ID}
gcloud --quiet deployment-manager deployments delete create-audit-logs-ds \
  --project ${AUDIT_PROJECT_ID} --delete-policy=ABANDON

echo "Audit project setup finished."
echo "Please remember to use \"--audit_project_id ${AUDIT_PROJECT_ID}\"" \
  "and \"--audit_dataset_id ${AUDIT_DATASET_ID}\" as audit inputs for other" \
  "scripts."
