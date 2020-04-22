terraform {
  backend "gcs" {}
}

provider "google" {
  version = "~> 3.12.0"
}

module "gcp-network" {
  source       = "terraform-google-modules/network/google"
  version      = "~> 2.0"
  project_id   = var.project_id
  network_name = var.network_name

  subnets = [
    {{range .SUBNETS}}
    {
      subnet_name   = "{{.NAME}}"
      subnet_ip     = "{{.IP_RANGE}}"
      subnet_region = var.region
      subnets_private_access = true
    },
    {{end}}
  ]

  secondary_ranges = {
  {{range .SUBNETS}}
    {{if get . "SECONDARY_RANGES"}}
    "{{.NAME}}" = [
      {{range .SECONDARY_RANGES}}
      {
        range_name    = "{{.NAME}}"
        ip_cidr_range = "{{.IP_RANGE}}"
      },
      {{end}}
    ],
    {{end}}
  {{end}}
  }
}
