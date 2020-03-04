include {
  path = find_in_parent_folders()
}

{{if not (index . "PARENT")}}
dependency "parent_folder" {
  config_path = "../../folder"

  mock_outputs = {
    name = "mock-folder"
  }
}

inputs = {
  parent = dependency.parent_folder.outputs.name
}
{{end}}
