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
#   - enables BigQuery and Google Cloud Storage services for holding data

set -u -e

print_usage() {
    echo "Usage:"
    echo "  create_team_project.sh \ "
    echo "    --owners_group <OWNERS_GROUP> \ "
    echo "    --users_group <USERS_GROUP> \ "
    echo "    --team_project_id <TEAM_PROJECT_ID> \ "
    echo "    --billing_account <BILLING_ACCOUNT> \ "
    echo "    --audit_project_id <AUDIT_PROJECT_ID> \ "
    echo "    --audit_dataset_id <AUDIT_DATASET_ID> \ "
    echo "    --start_vm [true|false]"
}

OWNERS_GROUP=""
USERS_GROUP=""
TEAM_PROJECT_ID=""
BILLING_ACCOUNT=""
AUDIT_PROJECT_ID=""
AUDIT_DATASET_ID=""
START_VM=false

while (( "$#" )); do
    if [[ $2 == --* ]]; then
      echo "Value of $1 starts with '--'. Missing value?"
      exit 1
    fi
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
    elif [[ $1 == "--start_vm" ]]; then
      if [[ $2 == "true" ]]; then
        START_VM=true
      fi
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
STATE_FILE="$0".state
# A list of state checkpoints for the script to resume to.
STATE_SET_PERMISSION="SET_PERMISSION"
STATE_SET_BILLING="SET_BILLING"
STATE_GRANT_ACCESS="GRANT_ACCESS"
STATE_CREATE_BUCKET="CREATE_BUCKET"
STATE_ENABLE_LOGGING="ENABLE_LOGGING"
STATE_ENABLE_SERVICES="ENABLE_SERVICES"
STATE_GRANT_TPU="GRANT_TPU"
STATE_IMPORT_DISK="IMPORT_DISK"
STATE_DEPLOY_VM="DEPLOY_VM"

if [[ ! -e ${STATE_FILE} ]]; then
  echo "Creating Google cloud project '${TEAM_PROJECT_ID}', which will be " \
  "used by '${USERS_GROUP}'."
  gcloud projects create "${TEAM_PROJECT_ID}"
  PROJECT_NUMBER=$(gcloud projects describe ${TEAM_PROJECT_ID} \
                          --format='value(projectNumber)')
  echo "Project created: ID=${TEAM_PROJECT_ID}, Number=${PROJECT_NUMBER}."
  echo ${STATE_SET_PERMISSION} > ${STATE_FILE}
else
  echo "Skip creating project since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_SET_PERMISSION} ]]; then
  PARENT_TYPE=$(gcloud projects describe ${TEAM_PROJECT_ID} \
                       --format='value(parent.type)')
  if [[ ${PARENT_TYPE} == "organization" ]]; then
    echo "Setting ${OWNERS_GROUP} as owner for project '${TEAM_PROJECT_ID}'."
    gcloud projects add-iam-policy-binding ${TEAM_PROJECT_ID} \
      --member="group:${OWNERS_GROUP}" --role="roles/owner"
    echo "Revoking individual owner access."
    USER_EMAIL=$(gcloud config list account --format "value(core.account)")
    gcloud projects remove-iam-policy-binding ${TEAM_PROJECT_ID} \
      --member="user:${USER_EMAIL}" --role="roles/owner"
  else
    echo "Because the project is created without an organization, it is not"
    echo "possible to use a group as an owner of the project. Granting"
    echo "${OWNERS_GROUP} permission to change IAM configuration, so that"
    echo "members of this group can add individual owners (including"
    echo "themselves) when needed."
    gcloud projects add-iam-policy-binding "${AUDIT_PROJECT_ID}" \
      --member="group:${OWNERS_GROUP}" \
      --role="roles/resourcemanager.projectIamAdmin"
  fi
  echo ${STATE_SET_BILLING} > ${STATE_FILE}
else
  echo "Skip setting permissions since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_SET_BILLING} ]]; then
  echo "Setting billing account ${BILLING_ACCOUNT} for project" \
    "'${TEAM_PROJECT_ID}'."
  gcloud beta billing projects link --billing-account "${BILLING_ACCOUNT}" \
    "${TEAM_PROJECT_ID}"
  echo ${STATE_GRANT_ACCESS} > ${STATE_FILE}
else
  echo "Skip setting billing since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_GRANT_ACCESS} ]]; then
  echo "Granting access to ${USERS_GROUP}..."
  # Permissions compute.instanceAdmin.v1 iam.serviceAccountActor needed for
  # SSH access to VMs if participants really need, but we don't grant it by
  # default.
  for role in viewer bigquery.user storage.objectCreator storage.objectViewer \
    ml.developer
  do
      echo "    - $role"
      gcloud projects add-iam-policy-binding "${TEAM_PROJECT_ID}" \
             --member="group:${USERS_GROUP}" \
             --role="roles/$role"
  done
  echo ${STATE_CREATE_BUCKET} > ${STATE_FILE}
else
  echo "Skip granting access since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_CREATE_BUCKET} ]]; then
  echo "Create a regular bucket for team file sharing..."
  STORAGE_CLASS=regional
  LOCATION=ASIA-SOUTHEAST1
  BUCKET_ID=${TEAM_PROJECT_ID}-shared-files
  gsutil mb -p ${TEAM_PROJECT_ID} -c ${STORAGE_CLASS} -l ${LOCATION} \
    gs://${BUCKET_ID}
  # legacyBucketWriter gives create/delete/update permissions.
  gsutil iam ch group:${USERS_GROUP}:legacyBucketWriter gs://${BUCKET_ID}
  echo ${STATE_ENABLE_LOGGING} > ${STATE_FILE}
else
  echo "Skip creating bucket since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_ENABLE_LOGGING} ]]; then
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
  echo ${STATE_ENABLE_SERVICES} > ${STATE_FILE}
else
  echo "Skip configuring logging since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_ENABLE_SERVICES} ]]; then
  echo "Enabling the following Google Cloud Platform Services"
  echo "  - Deployment Manager"
  echo "  - BigQuery"
  echo "  - Google Compute Engine"
  echo "  - Cloud Machine Learning Engine"
  gcloud services enable deploymentmanager bigquery compute ml.googleapis.com \
    --project ${TEAM_PROJECT_ID}
  echo ${STATE_GRANT_TPU} > ${STATE_FILE}
else
  echo "Skip enabling services since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_GRANT_TPU} ]]; then
  echo "Granting access to TPU service account..."
  CLOUD_ML_SERVICE_ACCOUNT=$(curl -H "Authorization: Bearer $(gcloud auth print-access-token)" https://ml.googleapis.com/v1/projects/${TEAM_PROJECT_ID}:getConfig)
  TPU_SERVICE_ACCOUNT=$(echo ${CLOUD_ML_SERVICE_ACCOUNT} | grep -oh -P '(?<=tpuServiceAccount":\s").+(?=")')
  gcloud projects add-iam-policy-binding "${TEAM_PROJECT_ID}" \
    --member="serviceAccount:${TPU_SERVICE_ACCOUNT}" \
    --role="roles/ml.serviceAgent"
  echo ${STATE_IMPORT_DISK} > ${STATE_FILE}
else
  echo "Skip enabling TPU since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_IMPORT_DISK} ]]; then
  echo 'Import boot disk from GCS bucket...'
  gcloud compute images create boot-disk --source-uri \
    gs://datathon-disks/boot-disk.tar.gz --project ${TEAM_PROJECT_ID}
  echo ${STATE_DEPLOY_VM} > ${STATE_FILE}
else
  echo "Skip importing boot disk since it has previously finished."
fi

if [[ `cat ${STATE_FILE}` == ${STATE_DEPLOY_VM} ]]; then
  echo "Deploying Google Cloud VM instances..."
  VM_NAME="work-machine"
  VM_ZONE="asia-southeast1-a"
  TEMP=`tempfile`
  cat <<EOF >>${TEMP}
resources:
- name: ${VM_NAME}
  type: compute.v1.instance
  properties:
    zone: ${VM_ZONE}
    machineType: https://www.googleapis.com/compute/v1/projects/${TEAM_PROJECT_ID}/zones/${VM_ZONE}/machineTypes/n1-standard-2
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
  if [ ${START_VM} = false ]; then
    gcloud compute instances stop ${VM_NAME} --zone ${VM_ZONE} \
      --project ${TEAM_PROJECT_ID}
  fi
  echo "Enabling OS loging for VM SSH access"
  gcloud compute project-info add-metadata \
    --metadata enable-oslogin=TRUE --project=${TEAM_PROJECT_ID}
else
  echo "Skip deploying VM since it has previously finished."
fi

rm -f ${STATE_FILE}

echo "Team project '${TEAM_PROJECT_ID}' setup finished."
