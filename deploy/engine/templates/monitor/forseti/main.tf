terraform {
  backend "gcs" {}
}

// TODO: stop doing this once 5.2.0 is released.
provider "google" {
  version = "~> 2.0"
}

// TODO: Incorporate latest features once 5.2.0 is released.
module "forseti" {
  source  = "terraform-google-modules/forseti/google"
  version = "~> 5.2.0"

  domain           = var.domain
  project_id       = var.project_id
  org_id           = var.org_id
  network          = "forseti-vpc"
  subnetwork       = "forseti-subnet"
  client_enabled   = false
  server_private   = true
  cloudsql_private = true
}
