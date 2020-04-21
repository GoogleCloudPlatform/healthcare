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

org_id                        = "{{.ORG_ID}}"
devops_project_id             = "{{.PROJECT_ID}}"
state_bucket                  = "{{.STATE_BUCKET}}"
repo_owner                    = "{{.REPO_OWNER}}"
repo_name                     = "{{.REPO_NAME}}"
branch_regex                  = "{{.BRANCH_REGEX}}"
{{- if index . "CONTINUOUS_DEPLOYMENT_ENABLED"}}
continuous_deployment_enabled = {{.CONTINUOUS_DEPLOYMENT_ENABLED}}
{{- end}}
{{- if index . "TRIGGER_ENABLED"}}
trigger_enabled               = {{.TRIGGER_ENABLED}}
{{- end}}
{{- if index . "TERRAFORM_ROOT"}}
terraform_root                = "{{.TERRAFORM_ROOT}}"
{{- end}}
{{- if index . "BUILD_VIEWERS"}}
build_viewers = [
  {{- range .BUILD_VIEWERS}}
  "{{.}}",
  {{- end}}
]
{{- end}}
