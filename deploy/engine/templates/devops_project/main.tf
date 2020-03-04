terraform {
  backend "gcs" {}
}

data "google_project" "project" {
  project_id = var.project_id
}

resource "google_resource_manager_lien" "lien" {
  parent       = "projects/${data.google_project.project.number}"
  restrictions = ["resourcemanager.projects.delete"]
  origin       = "managed-terraform"
  reason       = "Devops project used for managing infrastructure"
}
