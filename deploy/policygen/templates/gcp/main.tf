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

# This folder contains Terraform resources to configure GCP Organization Policies.
# (https://cloud.google.com/resource-manager/docs/organization-policy/org-policy-constraints)
# See the following resources for the details of policies enforced.

terraform {
  backend "gcs" {}
}

# App Engine
resource "google_organization_policy" "appengine_disable_code_download" {
  org_id     = var.org_id
  constraint = "appengine.disableCodeDownload"

  boolean_policy {
    enforced = true
  }
}

# Cloud SQL
resource "google_organization_policy" "sql_restrict_authorized_networks" {
  org_id     = var.org_id
  constraint = "sql.restrictAuthorizedNetworks"

  boolean_policy {
    enforced = true
  }
}

resource "google_organization_policy" "sql_restrict_public_ip" {
  org_id     = var.org_id
  constraint = "sql.restrictPublicIp"

  boolean_policy {
    enforced = true
  }
}

# Compute Engine
resource "google_organization_policy" "compute_disable_nested_virtualization" {
  org_id     = var.org_id
  constraint = "compute.disableNestedVirtualization"

  boolean_policy {
    enforced = true
  }
}

resource "google_organization_policy" "compute_disable_serial_port_access" {
  org_id     = var.org_id
  constraint = "compute.disableSerialPortAccess"

  boolean_policy {
    enforced = true
  }
}

resource "google_organization_policy" "compute_restrict_shared_vpc_host_projects" {
  count      = length(var.allowed_shared_vpc_host_projects) != 0 ? 1 : 0
  org_id     = var.org_id
  constraint = "compute.restrictSharedVpcHostProjects"

  list_policy {
    allow {
      values = var.allowed_shared_vpc_host_projects
    }
  }
}

resource "google_organization_policy" "compute_skip_default_network_creation" {
  org_id     = var.org_id
  constraint = "compute.skipDefaultNetworkCreation"

  boolean_policy {
    enforced = true
  }
}

resource "google_organization_policy" "compute_trusted_image_projects" {
  count      = length(var.allowed_trusted_image_projects) != 0 ? 1 : 0
  org_id     = var.org_id
  constraint = "compute.trustedImageProjects"

  list_policy {
    allow {
      values = var.allowed_trusted_image_projects
    }
  }
}

resource "google_organization_policy" "compute_vm_can_ip_forward" {
  org_id     = var.org_id
  constraint = "compute.vmCanIpForward"

  list_policy {
    deny {
      all = true
    }
  }
}

# Cloud Identity and Access Management
resource "google_organization_policy" "iam_allowed_policy_member_domains" {
  count      = length(var.allowed_policy_member_domains) != 0 ? 1 : 0
  org_id     = var.org_id
  constraint = "iam.allowedPolicyMemberDomains"

  list_policy {
    allow {
      values = var.allowed_policy_member_domains
    }
  }
}

# Resource Manager
resource "google_organization_policy" "compute_restrict_xpn_project_lien_removal" {
  org_id     = var.org_id
  constraint = "compute.restrictXpnProjectLienRemoval"

  boolean_policy {
    enforced = true
  }
}

# Google Cloud Platform - Resource Locations
resource "google_organization_policy" "gcp_resource_locations" {
  org_id     = var.org_id
  constraint = "gcp.resourceLocations"

  list_policy {
    allow {
      values = ["in:us-locations"]
    }
  }
}

# Cloud Storage
resource "google_organization_policy" "storage_uniform_bucket_level_access" {
  org_id     = var.org_id
  constraint = "storage.uniformBucketLevelAccess"

  boolean_policy {
    enforced = true
  }
}
