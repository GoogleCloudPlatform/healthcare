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

# This folder contains Terraform resources to setup CI/CD, which includes:
# - Necessary APIs to enable in the devops project for CI/CD purposes,
# - Necessary IAM permissions to set to enable Cloud Build Service Account perform CI/CD jobs.
# - Cloud Build Triggers to monitor GitHub repos to start CI/CD jobs.
#
# The Cloud Build configs can be found under the configs/ sub-folder.

# ***NOTE***: First follow
# https://cloud.google.com/cloud-build/docs/automating-builds/create-github-app-triggers#installing_the_cloud_build_app
# to install the Cloud Build app and connect your GitHub repository to your Cloud project.

terraform {
  backend "gcs" {
    bucket = "{{.STATE_BUCKET}}"
    prefix = "cicd"
  }
}

data "google_project" "devops" {
  project_id = var.devops_project_id
}

locals {
  devops_apis = [
    # TODO: Figure out how to use user_project_override and disable APIs in devops project
    # that are needed to obtain resource information in other projects.
    "bigquery.googleapis.com",
    "cloudbilling.googleapis.com",
    "cloudbuild.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "container.googleapis.com",
    "firebase.googleapis.com",
    "iam.googleapis.com",
    "servicenetworking.googleapis.com",
    "serviceusage.googleapis.com",
    "sqladmin.googleapis.com",
  ]
  cloudbuild_sa_viewer_roles = [
    "roles/browser",
    # Consider using viewer roles for individual services. But it is hard to know beforehand what
    # services are used in each project.
    "roles/viewer",
    "roles/iam.securityReviewer",
  ]
  cloudbuild_sa_editor_roles = concat(local.cloudbuild_sa_viewer_roles, [
    "roles/billing.user",
    "roles/compute.xpnAdmin",
    "roles/orgpolicy.policyAdmin",
    "roles/owner",
    "roles/resourcemanager.organizationAdmin",
    "roles/resourcemanager.folderCreator",
    "roles/resourcemanager.projectCreator",
  ])
}

locals {
  # Covert "" and "/" to "." in case users use them to indicate root of the git repo.
  terraform_root = trim((var.terraform_root == "" || var.terraform_root == "/") ? "." : var.terraform_root, "/")
  # ./ to indicate root is not recognized by Cloud Build Trigger.
  terraform_root_prefix = local.terraform_root == "." ? "" : "${local.terraform_root}/"
}

# Cloud Build - API
resource "google_project_service" "devops_apis" {
  for_each           = toset(local.devops_apis)
  project            = var.devops_project_id
  service            = each.value
  disable_on_destroy = false
}

# IAM permissions to allow approvers and contributors to view the build results.
resource "google_project_iam_member" "cloudbuild_viewers" {
  for_each = toset(var.build_viewers)
  project  = var.devops_project_id
  role     = "roles/cloudbuild.builds.viewer"
  member   = each.value
  depends_on = [
    google_project_service.devops_apis,
  ]
}

# Cloud Build - Cloud Build Service Account IAM permissions
# IAM permissions to allow Cloud Build SA to access state.
resource "google_storage_bucket_iam_member" "cloudbuild_state_iam" {
  bucket = var.state_bucket
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${data.google_project.devops.number}@cloudbuild.gserviceaccount.com"
  depends_on = [
    google_project_service.devops_apis,
  ]
}

# Grant Cloud Build Service Account access to the organization.
resource "google_organization_iam_member" "cloudbuild_sa_iam" {
  for_each = toset(var.continuous_deployment_enabled ? local.cloudbuild_sa_editor_roles : local.cloudbuild_sa_viewer_roles)
  org_id   = var.org_id
  role     = each.value
  member   = "serviceAccount:${data.google_project.devops.number}@cloudbuild.gserviceaccount.com"
  depends_on = [
    google_project_service.devops_apis,
  ]
}

# Cloud Build Triggers for CI.
resource "google_cloudbuild_trigger" "validate" {
  disabled = ! var.trigger_enabled
  provider = google-beta
  project  = var.devops_project_id
  name     = "tf-validate"

  included_files = [
    "${local.terraform_root_prefix}**",
  ]

  github {
    owner = var.repo_owner
    name  = var.repo_name
    pull_request {
      branch = var.branch_regex
    }
  }

  filename = "${local.terraform_root_prefix}cicd/configs/tf-validate.yaml"

  substitutions = {
    _TERRAFORM_ROOT = local.terraform_root
  }

  depends_on = [
    google_project_service.devops_apis,
  ]
}

resource "google_cloudbuild_trigger" "plan" {
  disabled = ! var.trigger_enabled
  provider = google-beta
  project  = var.devops_project_id
  name     = "tf-plan"

  included_files = [
    "${local.terraform_root_prefix}**",
  ]

  github {
    owner = var.repo_owner
    name  = var.repo_name
    pull_request {
      branch = var.branch_regex
    }
  }

  filename = "${local.terraform_root_prefix}cicd/configs/tf-plan.yaml"

  substitutions = {
    _TERRAFORM_ROOT = local.terraform_root
  }

  depends_on = [
    google_project_service.devops_apis,
  ]
}

# Cloud Build Triggers for CD.
resource "google_cloudbuild_trigger" "apply" {
  count    = var.continuous_deployment_enabled ? 1 : 0
  disabled = ! var.trigger_enabled
  provider = google-beta
  project  = var.devops_project_id
  name     = "tf-apply"

  included_files = [
    "${local.terraform_root_prefix}org/**",
    "${local.terraform_root_prefix}cicd/configs/tf-apply.yaml"
  ]

  github {
    owner = var.repo_owner
    name  = var.repo_name
    push {
      branch = var.branch_regex
    }
  }

  filename = "${local.terraform_root_prefix}cicd/configs/tf-apply.yaml"

  substitutions = {
    _TERRAFORM_ROOT = local.terraform_root
  }

  depends_on = [
    google_project_service.devops_apis,
  ]
}
