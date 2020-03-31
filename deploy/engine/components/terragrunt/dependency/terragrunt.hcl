{{range .DEPENDENCIES}}
dependency "{{.NAME}}" {
  config_path = "{{.PATH}}"

  mock_outputs = {
    {{range $k, $v := .MOCK_OUTPUTS}}
    {{$k}} = {{$v}}
    {{end}}
  }
}
{{end}}

inputs = {
  {{range $k, $v := .INPUTS}}
  {{$k}} = {{$v}}
  {{end}}
}
