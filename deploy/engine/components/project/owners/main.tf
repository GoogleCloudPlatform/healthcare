resource "google_project_iam_binding" "owners" {
  project = module.project.project_id
  role    = "roles/owner"
  members = var.owners
}
