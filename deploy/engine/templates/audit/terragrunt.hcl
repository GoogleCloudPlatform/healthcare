include {
  path = find_in_parent_folders()
}

dependencies {
  paths = [
    "../project-{{.PROJECT_ID}}/project",
  ]
}
