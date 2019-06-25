#!/bin/bash

source gbash.sh || exit 1

DEFINE_string project_yaml        "" "Location of the project config YAML."
DEFINE_string output_yaml_path    "" "Path to save a new YAML file environment variables substituted and generated fields populated."
DEFINE_string projects            "" "Project IDs within --project_yaml to deploy."
DEFINE_string output_rules_path   "" "Path to local directory or GCS bucket to output rules files."
DEFINE_bool   nodry_run           "false" "Actually create the projects."
gbash::init_google "$@"

set -x

if (( FLAGS_nodry_run )); then
    dry_run_flag="--nodry_run"
else
    dry_run_flag="--dry_run"
fi

other_flags=" --enable_new_style_resources "${dry_run_flag}

blaze run deploy:create_project -- \
    --project_yaml=${FLAGS_project_yaml} \
    --output_yaml_path=${FLAGS_output_yaml_path} \
    --output_rules_path=${FLAGS_output_rules_path} \
    --projects=${FLAGS_projects}\
    ${other_flags}

set +x
