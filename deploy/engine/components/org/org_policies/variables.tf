variable "org_id" {
  description = "The organization id"
  type        = string
}

variable "allowed_policy_member_domains" {
  description = "The list of G Suite Customer IDs corresponding to the allowed domains. If not specified, iam.allowedPolicyMemberDomains constraint will not be enforced"
  default     = []
}

variable "allowed_shared_vpc_host_projects" {
  description = "The list of allowed shared VPC host projects. Each entry must be specified in the form: under:organizations/ORGANIZATION_ID, under:folders/FOLDER_ID, or projects/PROJECT_ID. If not specified, compute.restrictSharedVpcHostProjects constraint will not be enforced"
  default     = []
}

variable "allowed_trusted_image_projects" {
  description = "The list of projects that can be used for image storage and disk instantiation for Compute Engine. Each entry must be specified in the form: projects/PROJECT_ID. If not specified, compute.trustedImageProjects constraint will not be enforced"
  default     = []
}
