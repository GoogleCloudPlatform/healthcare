variable "name" {
  type = string
}

variable "org_id" {
  type = string
}

variable "folder_id" {
  type    = string
  default = ""
}

variable "billing_account" {
  type = string
}

variable "apis" {
  type    = list(string)
  default = []
}

variable "enable_lien" {
  type    = bool
  default = true
}
