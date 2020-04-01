terraform {
  backend "gcs" {}
}

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
