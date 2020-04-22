variable "project_id" {
  description = "The GCP project id"
  type        = string
}

variable "region" {
  description = "The region to host the cluster in"
  default     = "us-central1"
  type        = string
}

variable "network_name" {
  description = "The name of the network to use for the cluste"
  type        = string
  default     = "default"
}

variable "cluster_name" {
  description = "The name of the GKE cluster"
  type        = string
}

variable "cluster_version" {
  description = "The Kubernetes version to use in the nodes and masters"
  type        = string
  default     = "latest"
}

variable "master_ipv4_cidr_block" {
  description = "The IP range in CIDR notation to use for the hosted master network"
  type        = string
}
