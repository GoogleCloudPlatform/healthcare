terraform {
  backend "gcs" {}
}

module "project" {
  source  = "terraform-google-modules/project-factory/google"
  version = "~> 7.0"

  name                    = var.name
  org_id                  = var.org_id
  folder_id               = var.folder_id
  billing_account         = var.billing_account
  lien                    = var.enable_lien
  activate_apis           = var.apis
  default_service_account = "keep"
  skip_gcloud_download    = true
}
