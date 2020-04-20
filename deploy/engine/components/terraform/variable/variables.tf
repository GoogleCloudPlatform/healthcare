variable "{{.NAME}}" {
  type = {{.TYPE}}
  {{- if index . "DEFAULT"}}
  default = {{.DEFAULT}}
  {{- end}}
}
