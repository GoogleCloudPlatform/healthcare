include {
  path = find_in_parent_folders()
}

{{if index . "PARENT_TYPE" -}}
{{if eq .PARENT_TYPE "FOLDER"}}
dependency "parent_folder" {
  config_path = "../../folder"

  mock_outputs = {
    name = "mock-folder"
  }
}

inputs = {
  folder_id = dependency.parent_folder.outputs.name
}
{{end}}
{{end}}