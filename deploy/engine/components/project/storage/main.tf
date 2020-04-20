module "{{resourceName .NAME}}" {
  source  = "terraform-google-modules/cloud-storage/google//modules/simple_bucket"
  version = "~> 1.4"

  name       = "{{.NAME}}"
  project_id = var.project_id
  location   = var.storage_location
  {{- if index . "IAM_MEMBERS"}}
  iam_members = [
    {{- range .IAM_MEMBERS}}
    {
      role   = "{{.ROLE}}"
      member = "{{.MEMBER}}"
    },
    {{- end}}
  ]
  {{- end}}
}
