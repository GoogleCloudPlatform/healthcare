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

org_id = "{{.ORG_ID}}"
{{- if index . "ALLOWED_POLICY_MEMBER_DOMAINS"}}
allowed_policy_member_domains = [
  {{- range .ALLOWED_POLICY_MEMBER_DOMAINS}}
  "{{.}}",
  {{- end}}
]
{{- end}}
{{- if index . "ALLOWED_SHARED_VPC_HOST_PROJECTS"}}
allowed_shared_vpc_host_projects = [
  {{- range .ALLOWED_SHARED_VPC_HOST_PROJECTS}}
  "{{.}}",
  {{- end}}
]
{{- end}}
{{- if index . "ALLOWED_TRUSTED_IMAGE_PROJECTS"}}
allowed_trusted_image_projects = [
  {{- range .ALLOWED_TRUSTED_IMAGE_PROJECTS}}
  "{{.}}",
  {{- end}}
]
{{- end}}
