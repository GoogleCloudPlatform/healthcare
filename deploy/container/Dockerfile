# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM gcr.io/cloud-builders/wget as builder

ARG TERRAFORM_VERSION
ARG TERRAFORM_SHA256SUM
ARG TERRAGRUNT_VERSION
ARG TERRAGRUNT_SHA256SUM

WORKDIR /builder/tmp

RUN apt-get update && \
      apt-get -y install unzip

RUN wget -q https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip && \
      echo "${TERRAFORM_SHA256SUM} terraform_${TERRAFORM_VERSION}_linux_amd64.zip" | sha256sum -c && \
      unzip terraform_${TERRAFORM_VERSION}_linux_amd64.zip

RUN wget -q https://github.com/gruntwork-io/terragrunt/releases/download/v${TERRAGRUNT_VERSION}/terragrunt_linux_amd64 && \
      echo "${TERRAGRUNT_SHA256SUM} terragrunt_linux_amd64" | sha256sum -c

FROM marketplace.gcr.io/google/ubuntu1804

COPY --from=builder /builder/tmp/terraform /usr/local/bin/terraform
COPY --from=builder /builder/tmp/terragrunt_linux_amd64 /usr/local/bin/terragrunt

RUN chmod +x /usr/local/bin/terraform && \
      chmod +x /usr/local/bin/terragrunt

ENTRYPOINT terraform
