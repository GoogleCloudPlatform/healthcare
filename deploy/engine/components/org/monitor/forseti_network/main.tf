terraform {
  backend "gcs" {}
}

module "network" {
  source = "terraform-google-modules/network/google"
  version = "~> 2.1"

  project_id              = var.project
  network_name            = "forseti-vpc"
  subnets                 = [{
    subnet_name   = "forseti-subnet"
    subnet_ip     = "10.10.10.0/24"
    subnet_region = "us-central1"
  }]
}

module "router" {
  source  = "terraform-google-modules/cloud-router/google"
  version =  "~> 0.1"

  name    = "forseti-router"
  project = var.project
  region  = "us-central1"
  network = module.network.network_name

  nats = [{
    name = "forseti-nat"
  }]
}
