{{- range .DEPENDENCIES}}
dependency "{{.NAME}}" {
  config_path = "{{.PATH}}"

  {{- if not (get . "MOCK_OUTPUTS")}}
  skip_outputs = true
  {{- end}}

  {{- if index . "MOCK_OUTPUTS"}}
  mock_outputs = {
    {{- range $k, $v := .MOCK_OUTPUTS}}
    {{$k}} = {{$v}}
    {{- end}}
  }
  {{- end}}
}
{{end}}

{{- if index . "INPUTS"}}
inputs = {
  {{- range $k, $v := .INPUTS}}
  {{$k}} = {{$v}}
  {{- end}}
}
{{- end}}
