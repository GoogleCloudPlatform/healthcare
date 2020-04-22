variable "project_id" {
  description = "The GCP project id"
  type        = string
}

variable "region" {
  description = "The region where the subnets will be created"
  default     = "us-central1"
  type        = string
}

variable "network_name" {
  description = "The name of the network"
  type        = string
}
