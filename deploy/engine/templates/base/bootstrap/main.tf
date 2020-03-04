resource "google_project" "devops_project" {
  project_id          = var.project_id
  name                = var.project_id
  org_id              = var.org_id
  billing_account     = var.billing_account
  auto_create_network = false
}
