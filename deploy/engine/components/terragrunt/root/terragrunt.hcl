remote_state {
  backend = "gcs"
  config = {
    bucket = "{{.STATE_BUCKET}}"

    prefix = "${path_relative_to_include()}"
    project = "{{.PROJECT_ID}}"
    location = "us-central1"
  }
}
