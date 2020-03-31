resource "google_project_iam_binding" "owners" {
  project = var.project_id
  role    = "roles/owner"
  members = var.owners
}
