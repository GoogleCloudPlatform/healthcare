#/bin/bash

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

# This script sets up a work environment project for a datathon team or similar
# applications. It creates
#   - creates a VM with RStudio server running on it
#   - enables BigQuery and Google Cloud Storage services for holding data move as much of this setup as possible to deployment manager
# script.

set -u -e

print_usage() {
    echo "Usage:"
    echo "  create_team_project.sh \ "
    echo "    --owners_group <OWNERS_GROUP> \ "
    echo "    --users_group <USERS_GROUP> \ "
    echo "    --team_project_id <TEAM_PROJECT_ID> \ "
    echo "    --billing_account <BILLING_ACCOUNT> \ "
    echo "    --audit_project_id <AUDIT_PROJECT_ID> \ "
    echo "    --audit_dataset_id <AUDIT_DATASET_ID>"
}

OWNERS_GROUP=""
USERS_GROUP=""
TEAM_PROJECT_ID=""
BILLING_ACCOUNT=""
AUDIT_PROJECT_ID=""
AUDIT_DATASET_ID=""

while (( "$#" )); do
    if [[ $1 == "--owners_group" ]]; then
        OWNERS_GROUP=$2
    elif [[ $1 == "--users_group" ]]; then
        USERS_GROUP=$2
    elif [[ $1 == "--team_project_id" ]]; then
        TEAM_PROJECT_ID=$2
    elif [[ $1 == "--billing_account" ]]; then
        BILLING_ACCOUNT=$2
    elif [[ $1 == "--audit_project_id" ]]; then
        AUDIT_PROJECT_ID=$2
    elif [[ $1 == "--audit_dataset_id" ]]; then
        AUDIT_DATASET_ID=$2
    else
        echo "Unknown flag $1"
        exit 1
    fi
    shift 2
done

if [[ -z ${OWNERS_GROUP} ]] || [[ -z ${USERS_GROUP} ]] || \
       [[ -z ${TEAM_PROJECT_ID} ]] || [[ -z ${BILLING_ACCOUNT} ]] || \
       [[ -z ${AUDIT_PROJECT_ID} ]] || [[ -z ${AUDIT_DATASET_ID} ]]; then
    print_usage
    exit 1
fi

USER_EMAIL=$(gcloud config list account --format "value(core.account)")

echo "Creating Google cloud project '${TEAM_PROJECT_ID}', which will be used " \
     "by '${USERS_GROUP}'."
gcloud projects create "${TEAM_PROJECT_ID}"

PROJECT_NUMBER=$(gcloud projects describe ${TEAM_PROJECT_ID} \
                        --format='value(projectNumber)')
echo "Project created: ID=${TEAM_PROJECT_ID}, Number=${PROJECT_NUMBER}."

PARENT_TYPE=$(gcloud projects describe ${TEAM_PROJECT_ID} \
                     --format='value(parent.type)')
if [[ ${PARENT_TYPE} == "organization" ]]; then
  echo "Setting ${OWNERS_GROUP} as owner for project '${TEAM_PROJECT_ID}'."
  gcloud projects add-iam-policy-binding ${TEAM_PROJECT_ID} \
    --member="group:${OWNERS_GROUP}" --role="roles/owner"
  echo "Revoking individual owner access."
  gcloud projects remove-iam-policy-binding ${TEAM_PROJECT_ID} \
    --member="user:${USER_EMAIL}" --role="roles/owner"
else
  echo "Skipping step for setting ${OWNERS_GROUP} as the owner of the project."
  echo "This is because we have not set up an organization for datathons."
fi

echo "Setting up billing..."
gcloud beta billing projects link --billing-account "${BILLING_ACCOUNT}" \
       "${TEAM_PROJECT_ID}"

echo "Granting access to ${USERS_GROUP}..."
# Permissions compute.instanceAdmin.v1 iam.serviceAccountActor needed for SSH
# access to VMs if participants really need, but we don't grant it by default.
for role in viewer bigquery.user storage.objectCreator storage.objectViewer
do
    echo "    - $role"
    gcloud projects add-iam-policy-binding "${TEAM_PROJECT_ID}" \
           --member="group:${USERS_GROUP}" \
           --role="roles/$role"
done

echo "Create a regular bucket for team file sharing..."
STORAGE_CLASS=regional
LOCATION=ASIA-SOUTHEAST1
BUCKET_ID=${TEAM_PROJECT_ID}-shared-files
gsutil mb -p ${TEAM_PROJECT_ID} -c ${STORAGE_CLASS} -l ${LOCATION} \
       gs://${BUCKET_ID}

echo "Enabling audit logging for the project."
SINK_NAME=audit-logs-to-bigquery
AUDIT_DATASET_URL="bigquery.googleapis.com/projects/${AUDIT_PROJECT_ID}/datasets/${AUDIT_DATASET_ID}"
LOG_SERVICE_ACCT=$((gcloud --quiet logging sinks create "${SINK_NAME}" \
  "${AUDIT_DATASET_URL}" --project ${TEAM_PROJECT_ID}) 2>&1 \
  | grep 'Please remember to grant' | \
  sed -e 's/^[^`]*`serviceAccount://' -e 's/`.*$//')

# Add the log service account as WRITER to the audit BigQuery dataset.
TEMP=`tempfile`
bq show --format=prettyjson "${AUDIT_PROJECT_ID}:${AUDIT_DATASET_ID}" | jq \
  ".access+[{\"userByEmail\":\"${LOG_SERVICE_ACCT?}\",\"role\":\"WRITER\"}]" \
  | jq '{access:.}' > ${TEMP}
bq update --source ${TEMP} "${AUDIT_PROJECT_ID}:${AUDIT_DATASET_ID}"


echo "Enabling the following Google Cloud Platform Services"
echo "  - Deployment Manager"
echo "  - BigQuery"
echo "  - Google Compute Engine"
echo "  - Cloud Machine Learning Engine"
gcloud services enable deploymentmanager bigquery compute ml.googleapis.com \
       --project ${TEAM_PROJECT_ID}

echo "Granting access to TPU service account..."
CLOUD_ML_SERVICE_ACCOUNT=$(curl -H "Authorization: Bearer $(gcloud auth print-access-token)" https://ml.googleapis.com/v1/projects/${TEAM_PROJECT_ID}:getConfig)
TPU_SERVICE_ACCOUNT=$(echo ${CLOUD_ML_SERVICE_ACCOUNT} | grep -oh -P '(?<=tpuServiceAccount":\s").+(?=")')
gcloud projects add-iam-policy-binding "${TEAM_PROJECT_ID}" \
       --member="serviceAccount:${TPU_SERVICE_ACCOUNT}" \
       --role="roles/ml.serviceAgent"

echo 'Import boot disk from GCS bucket...'
gcloud compute images create boot-disk --source-uri \
       gs://datathon-disks/rstudio-boot.tar.gz --project ${TEAM_PROJECT_ID}

echo "Deploying Google Cloud VM instances..."
TEMP=`tempfile`
cat <<EOF >>${TEMP}
resources:
- name: work-machine
  type: compute.v1.instance
  properties:
    zone: asia-southeast1-a
    machineType: https://www.googleapis.com/compute/v1/projects/${TEAM_PROJECT_ID}/zones/asia-southeast1-a/machineTypes/n1-standard-2
    disks:
    - deviceName: boot
      type: PERSISTENT
      boot: true
      autoDelete: true
      initializeParams:
        sourceImage: https://www.googleapis.com/compute/v1/projects/${TEAM_PROJECT_ID}/global/images/boot-disk
    networkInterfaces:
    - network: https://www.googleapis.com/compute/v1/projects/${TEAM_PROJECT_ID}/global/networks/default
      accessConfigs:
      - name: External NAT
        type: ONE_TO_ONE_NAT
- name: firewall-allow-rstudio
  type: compute.v1.firewall
  properties:
    allowed:
    - IPProtocol: tcp
      ports:
      - 8787
    sourceRanges: [0.0.0.0/0]
EOF
gcloud deployment-manager deployments create work-vm-setup \
       --config ${TEMP} --project ${TEAM_PROJECT_ID}
gcloud --quiet deployment-manager deployments delete work-vm-setup \
       --project ${TEAM_PROJECT_ID} --delete-policy=ABANDON

echo "Team project '${TEAM_PROJECT_ID}' setup finished."
