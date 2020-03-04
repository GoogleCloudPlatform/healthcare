#!/bin/bash

set -e

PROJECT={{.DEVOPS_PROJECT_ID}}

IMPORT=false

while getopts "i" o; do
    case "${o}" in
        i)
            IMPORT=true
            ;;
        *)
            echo Invalid flag: ${o}
            exit 1
            ;;
    esac
done

terraform init

if ${IMPORT}; then
  terraform import google_project.devops_project ${PROJECT}
fi

terraform apply
