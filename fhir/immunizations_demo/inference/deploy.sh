#!/bin/bash
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# A script for deploying to Cloud Functions, example usage:
# ./deploy.sh --name demo \
#             --topic demo-topic \
#             --env_vars MODEL=devdaysdemo,VERSION=v1

set -u -e

print_usage() {
    echo "Please run this script from the same directory."
    echo "Usage:"
    echo "  ./deploy.sh \ "
    echo "    --name <FUNCTION_NAME> \ "
    echo "    --topic <TRIGGER_TOPIC> \ "
    echo "    --env_vars <ENVIRONMENT_VARIABLES>"
}

NAME=""
TOPIC=""
ENV_VARS=""
UTILS=utils.py

while (( "$#" )); do
    if [[ $2 == --* ]]; then
      echo "Value of $1 starts with '--'. Missing value?"
      exit 1
    fi
    if [[ $1 == "--name" ]]; then
      NAME=$2
    elif [[ $1 == "--topic" ]]; then
      TOPIC=$2
    elif [[ $1 == "--env_vars" ]]; then
      ENV_VARS=$2
    else
      echo "Unknown flag $1"
      exit 1
    fi
    shift 2
done

if [[ -z ${NAME} ]] || [[ -z ${TOPIC} ]] || [[ -z ${ENV_VARS} ]]; then
    print_usage
    exit 1
fi

# Copy the helper file over.
cp ../shared/${UTILS} .

# Deploy to Cloud Functions.
gcloud beta functions deploy ${NAME} --runtime python37 \
    --entry-point main --trigger-topic ${TOPIC} \
    --set-env-vars ${ENV_VARS}

# Cleanup.
rm ${UTILS}
