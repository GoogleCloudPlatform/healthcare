terraform {
  backend "gcs" {}
}

resource "google_organization_iam_audit_config" "config" {
  org_id  = var.org_id
  service = "allServices"

  audit_log_config {
    log_type = "DATA_READ"
  }
  audit_log_config {
    log_type = "DATA_WRITE"
  }
  audit_log_config {
    log_type = "ADMIN_READ"
  }
}

module "bigquery_log_export" {
  source  = "terraform-google-modules/log-export/google"
  version = "~> 4.0"

  log_sink_name        = "bigquery-org-sink"
  parent_resource_type = "organization"
  parent_resource_id   = var.org_id
  filter               = "logName:\"logs/cloudaudit.googleapis.com\""
  destination_uri      = "bigquery.googleapis.com/projects/${var.project_id}/datasets/${var.dataset_name}"
}

# NOTE: The current CFT bigquery destination will give the running user OWNER access to the dataset.
# This will be fixed in v5.
module "bigquery_destination" {
  source  = "terraform-google-modules/log-export/google//modules/bigquery"
  version = "~> 4.0"

  project_id                  = var.project_id
  dataset_name                = var.dataset_name
  default_table_expiration_ms = 365 * 8.64 * pow(10, 7) # 365 days
  log_sink_writer_identity    = module.bigquery_log_export.writer_identity
}

module "storage_log_export" {
  source  = "terraform-google-modules/log-export/google"
  version = "~> 4.0"

  log_sink_name        = "storage-org-sink"
  parent_resource_type = "organization"
  parent_resource_id   = var.org_id
  filter               = "logName:\"logs/cloudaudit.googleapis.com\""
  destination_uri      = module.storage_destination.destination_uri
}

module "storage_destination" {
  source  = "terraform-google-modules/log-export/google//modules/storage"
  version = "~> 4.0"

  project_id               = var.project_id
  storage_bucket_name      = var.bucket_name
  storage_class            = "COLDLINE"
  # object_expiration_days = 7 * 365 # 7 years -- need to add support for this
  log_sink_writer_identity = module.storage_log_export.writer_identity
}

resource "google_organization_iam_member" "security_reviewer_auditors" {
  org_id = var.org_id
  role   = "roles/iam.securityReviewer"
  member = var.auditors
}

# TODO: grant reader access on audit resources to auditors group.
