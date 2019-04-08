# Copyright 2019 Google LLC
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
# App Engine backend to import audit data into the HCLS Audit API.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@deploy_deps//:requirements.bzl", "requirement")

py_library(
    name = "audit_api_lib",
    srcs = [
        "audit_api_client.py",
        "audit_api_resources.py",
    ],
)

py_library(
    name = "gcs_client_lib",
    srcs = ["gcs_client.py"],
    deps = [
        "//third_party/py/google/cloud:core_future",
        "//third_party/py/google/cloud/storage:storage_future",
    ],
)

py_library(
    name = "project_config_processor_lib",
    srcs = ["project_config_processor.py"],
    deps = [
        requirement("pyyaml"),
        "//third_party/py/google/cloud:core_future",
    ],
)

py_test(
    name = "audit_api_client_test",
    srcs = ["audit_api_client_test.py"],
    python_version = "PY3",
    deps = [
        requirement("absl-py"),
        ":audit_api_lib",
    ],
)

py_test(
    name = "gcs_client_test",
    srcs = ["gcs_client_test.py"],
    python_version = "PY3",
    deps = [
        requirement("absl-py"),
        requirement("mock"),
        ":gcs_client_lib",
        "//third_party/py/google/cloud:core_future",
        "//third_party/py/google/cloud/storage:storage_future",
    ],
)

py_test(
    name = "project_config_processor_test",
    srcs = ["project_config_processor_test.py"],
    python_version = "PY3",
    deps = [
        requirement("absl-py"),
        requirement("mock"),
        requirement("pyyaml"),
        ":audit_api_lib",
        ":gcs_client_lib",
        ":project_config_processor_lib",
        "//third_party/py/google/cloud:core_future",
        "//third_party/py/google/cloud/storage:storage_future",
    ],
)
