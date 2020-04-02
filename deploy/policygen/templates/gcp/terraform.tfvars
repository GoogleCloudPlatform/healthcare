org_id       = "{{.ORG_ID}}"
{{if index . "ALLOWED_POLICY_MEMBER_DOMAINS"}}
allowed_policy_member_domains = [
  {{range .ALLOWED_POLICY_MEMBER_DOMAINS}}
  "{{.}}",
  {{end}}
]
{{end}}
{{if index . "ALLOWED_SHARED_VPC_HOST_PROJECTS"}}
allowed_shared_vpc_host_projects = [
  {{range .ALLOWED_SHARED_VPC_HOST_PROJECTS}}
  "{{.}}",
  {{end}}
]
{{end}}
{{if index . "ALLOWED_TRUSTED_IMAGE_PROJECTS"}}
allowed_trusted_image_projects = [
  {{range .ALLOWED_TRUSTED_IMAGE_PROJECTS}}
  "{{.}}",
  {{end}}
]
{{end}}