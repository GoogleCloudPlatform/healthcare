terraform {
  backend "gcs" {}
}

provider "google" {
  version = "~> 3.12.0"
}

provider "google-beta" {
  version = "~> 3.12.0"
}

# These are defined the same way in the network component.
locals {
  subnet_name             = "${var.cluster_name}-subnet"
  master_auth_subnet_name = "${var.cluster_name}-master-subnet"
  pods_range_name         = "${var.cluster_name}-ip-range-pods"
  svc_range_name          = "${var.cluster_name}-ip-range-svc"
}


# From
# https://github.com/terraform-google-modules/terraform-google-kubernetes-engine/tree/master/modules/safer-cluster-update-variant
module "gke" {
  source                         = "terraform-google-modules/kubernetes-engine/google//modules/safer-cluster"

  # Required
  name                           = var.cluster_name
  kubernetes_version             = var.cluster_version
  project_id                     = var.project_id
  region                         = var.region
  regional                       = true
  network                        = var.network_name
  subnetwork                     = local.subnet_name
  ip_range_pods                  = local.pods_range_name
  ip_range_services              = local.svc_range_name
  master_ipv4_cidr_block         = var.master_ipv4_cidr_block

  # Optional
  # Some of these were taken from the example config at
  # https://github.com/terraform-google-modules/terraform-google-kubernetes-engine/tree/master/examples/safer_cluster
  istio    = true
  skip_provisioners = true
  grant_registry_access = true

  # Need to either disable private endpoint, or enable master auth networks.
  {{if get . "MASTER_AUTHORIZED_NETWORKS"}}
  master_authorized_networks = [
    {{range .MASTER_AUTHORIZED_NETWORKS }}
    {
      cidr_block   = "{{.CIDR_BLOCK}}"
      display_name = "{{.DISPLAY_NAME}}"
    },
    {{end}}
  ]
  {{else}}
  enable_private_endpoint = false
  {{end}}
}
