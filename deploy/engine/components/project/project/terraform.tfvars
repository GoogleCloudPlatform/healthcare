name            = "{{.PROJECT_ID}}"
org_id          = "{{.ORG_ID}}"
billing_account = "{{.BILLING_ACCOUNT}}"
{{if index . "ENABLE_LIEN"}}
enable_lien     = {{.ENABLE_LIEN}}
{{end}}
{{if index . "APIS"}}
apis = [
  {{range .APIS}}
  "{{.}}",
  {{end}}
]
{{end}}