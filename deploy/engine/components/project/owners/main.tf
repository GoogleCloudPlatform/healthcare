
# Project level IAM permissions for project owners.
resource "google_project_iam_binding" "owners" {
  project = module.project.project_id
  role    = "roles/owner"
  members = var.owners
}
