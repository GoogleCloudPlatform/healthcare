terraform {
  backend "gcs" {}
}

resource "google_folder" "folder" {
  display_name = var.display_name
  parent       = var.parent
}
