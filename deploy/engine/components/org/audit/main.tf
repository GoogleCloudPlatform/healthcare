# Copyright 2020 Google Inc.
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

# This folder contains Terraform resources related to audit, which includes:
# - Organization IAM Audit log configs (https://cloud.google.com/logging/docs/audit),
# - BigQuery log sink creation and configuration for short term log storage,
# - Cloud Storage log sink creation and configuration for long term log storage,
# - IAM permissions to grant log Auditors iam.securityReviewer role to view the logs.

terraform {
  backend "gcs" {}
}

# IAM Audit log configs to enable collection of all possible audit logs.
resource "google_organization_iam_audit_config" "config" {
  org_id  = var.org_id
  service = "allServices"

  audit_log_config {
    log_type = "DATA_READ"
  }
  audit_log_config {
    log_type = "DATA_WRITE"
  }
  audit_log_config {
    log_type = "ADMIN_READ"
  }
}

# BigQuery log sink.
module "bigquery_log_export" {
  source  = "terraform-google-modules/log-export/google"
  version = "~> 4.0"

  log_sink_name        = "bigquery-org-sink"
  parent_resource_type = "organization"
  parent_resource_id   = var.org_id
  include_children     = true
  filter               = "logName:\"logs/cloudaudit.googleapis.com\""
  destination_uri      = "bigquery.googleapis.com/projects/${var.project_id}/datasets/${module.bigquery_destination.bigquery_dataset.dataset_id}"
}

# TODO: replace with terraform-google-modules/log-export/google//modules/bigquery
# once https://github.com/terraform-google-modules/terraform-google-log-export/pull/52 is merged.
module "bigquery_destination" {
  source  = "terraform-google-modules/bigquery/google"
  version = "~> 4.0"

  dataset_id                  = var.dataset_name
  project_id                  = var.project_id
  location                    = "us-east1"
  default_table_expiration_ms = 365 * 8.64 * pow(10, 7) # 365 days
  access = [
    {
      role          = "roles/bigquery.dataOwner",
      special_group = "projectOwners"
    },
    {
      role           = "roles/bigquery.dataViewer",
      group_by_email = split(":", var.auditors)[1] // trim 'group:' prefix
    },
  ]
}

resource "google_project_iam_member" "bigquery_sink_member" {
  project = module.bigquery_destination.bigquery_dataset.project
  role    = "roles/bigquery.dataEditor"
  member  = module.bigquery_log_export.writer_identity
}

# Cloud Storage log sink.
module "storage_log_export" {
  source  = "terraform-google-modules/log-export/google"
  version = "~> 4.0"

  log_sink_name        = "storage-org-sink"
  parent_resource_type = "organization"
  parent_resource_id   = var.org_id
  include_children     = true
  filter               = "logName:\"logs/cloudaudit.googleapis.com\""
  destination_uri      = "storage.googleapis.com/${module.storage_destination.bucket.name}"
}

# TODO: Replace with terraform-google-modules/log-export/google//modules/storage
# once https://github.com/terraform-google-modules/terraform-google-log-export/pull/52  is fixed.
module "storage_destination" {
  source  = "terraform-google-modules/cloud-storage/google//modules/simple_bucket"
  version = "~> 1.4"

  name          = var.bucket_name
  project_id    = var.project_id
  location      = "us-central1"
  storage_class = "COLDLINE"

  lifecycle_rules = [{
    action = {
      type = "Delete"
    }
    condition = {
      age        = 7 * 365 # 7 years
      with_state = "ANY"
    }
  }]

  iam_members = [
    {
      role   = "roles/storage.objectViewer"
      member = var.auditors
    },
  ]
}

# Deploy sink member as separate resource otherwise Terraform will return error:
# `The "for_each" value depends on resource attributes that cannot be determined
# until apply, so Terraform cannot predict how many instances will be created.`
resource "google_storage_bucket_iam_member" "storage_sink_member" {
  bucket = module.storage_destination.bucket.name
  role   = "roles/storage.objectCreator"
  member = module.storage_log_export.writer_identity
}

# IAM permissions to grant log Auditors iam.securityReviewer role to view the logs.
resource "google_organization_iam_member" "security_reviewer_auditors" {
  org_id = var.org_id
  role   = "roles/iam.securityReviewer"
  member = var.auditors
}
