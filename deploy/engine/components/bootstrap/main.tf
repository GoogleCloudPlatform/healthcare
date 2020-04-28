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

# This folder contains Terraform resources to setup the devops project, which includes:
# - The project itself,
# - APIs to enable,
# - Deletion lien,
# - Project level IAM permissions for the project owners,
# - A Cloud Storage bucket to store Terraform states for all deployments,
# - Org level IAM permissions for org admins.

// TODO: replace with https://github.com/terraform-google-modules/terraform-google-bootstrap
{{- if enabled . "BOOTSTRAP_GCS_BACKEND"}}
terraform {
  backend "gcs" {
    bucket = "{{.STATE_BUCKET}}"
    prefix = "bootstrap"
  }
}
{{- end}}

# Create the project, enable APIs, and create the deletion lien, if specified.
module "project" {
  source  = "terraform-google-modules/project-factory/google"
  version = "~> 7.0"

  name                    = var.devops_project_id
  org_id                  = var.org_id
  billing_account         = var.billing_account
  lien                    = true
  default_service_account = "keep"
  skip_gcloud_download    = true
  activate_apis = [
    "cloudbuild.googleapis.com",
    "secretmanager.googleapis.com",
  ]
}

# Terraform state bucket, hosted in the devops project.
module "state_bucket" {
  source  = "terraform-google-modules/cloud-storage/google//modules/simple_bucket"
  version = "~> 1.4"

  name       = var.state_bucket
  project_id = module.project.project_id
  location   = var.storage_location
}

# Project level IAM permissions for devops project owners.
resource "google_project_iam_binding" "devops_owners" {
  project = module.project.project_id
  role    = "roles/owner"
  members = var.devops_owners
}

# Org level IAM permissions for org admins.
resource "google_organization_iam_member" "org_admin" {
  org_id = var.org_id
  role   = "roles/resourcemanager.organizationAdmin"
  member = var.org_admin
}
