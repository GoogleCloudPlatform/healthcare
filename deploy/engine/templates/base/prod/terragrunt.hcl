remote_state {
  backend = "gcs"
  config = {
    bucket = "{{.STATE_BUCKET}}"

    prefix = "${path_relative_to_include()}"
    project = "{{.DEVOPS_PROJECT_ID}}"
    location = "us-central1"
  }
}
