terraform {
  backend "gcs" {}
}

module "project" {
  source  = "terraform-google-modules/project-factory/google"
  version = "~> 6.0"

  name                    = var.name
  org_id                  = var.org_id
  folder_id               = var.folder_id
  billing_account         = var.billing_account
  lien                    = true
  default_service_account = "keep"
}
