# Copyright 2020 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

terraform {
  backend "gcs" {}
}

module "network" {
  source  = "terraform-google-modules/network/google"
  version = "~> 2.1"

  project_id   = var.project
  network_name = "forseti-vpc"
  subnets = [{
    subnet_name   = "forseti-subnet"
    subnet_ip     = "10.10.10.0/24"
    subnet_region = "us-central1"
  }]
}

module "router" {
  source  = "terraform-google-modules/cloud-router/google"
  version = "~> 0.1"

  name    = "forseti-router"
  project = var.project
  region  = "us-central1"
  network = module.network.network_name

  nats = [{
    name = "forseti-nat"
  }]
}
