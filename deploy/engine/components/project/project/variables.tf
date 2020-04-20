# Copyright 2020 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
