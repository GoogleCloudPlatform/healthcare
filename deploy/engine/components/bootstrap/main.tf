// TODO: replace with https://github.com/terraform-google-modules/terraform-google-bootstrap

# TODO: uncomment this block after the initial deployment and run
# `terraform init` again to back up the state to its own state bucket.
#
# terraform {
#   backend "gcs" {
#     bucket = "{{.STATE_BUCKET}}"
#     prefix = "bootstrap"
#   }
# }

module "project" {
  source  = "terraform-google-modules/project-factory/google"
  version = "~> 7.0"

  name                    = var.project_id
  org_id                  = var.org_id
  billing_account         = var.billing_account
  lien                    = true
  default_service_account = "keep"
  skip_gcloud_download    = true
}

module "state_bucket" {
  source  = "terraform-google-modules/cloud-storage/google//modules/simple_bucket"
  version = "~> 1.4"

  name       = var.state_bucket
  project_id = var.project_id
  location   = var.storage_location
}
