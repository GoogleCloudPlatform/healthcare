owners = [
  {{- range .PROJECT_OWNERS}}
  "{{.}}",
  {{- end}}
]
