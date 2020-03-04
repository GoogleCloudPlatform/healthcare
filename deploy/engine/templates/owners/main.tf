resource "google_project_iam_binding" "owners" {
  {{if index . "PROJECT_ID" -}}
  project = var.project_id
  {{else -}}
  project = module.project.project_id
  {{end -}}
  role    = "roles/owner"
  members = var.owners
}
