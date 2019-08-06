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

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

go_library(
    name = "processors",
    srcs = [
        "audit_api_client.go",
        "gcs_client.go",
        "processor.go",
    ],
    deps = ["//third_party/golang/cloud/storage"],
)

go_test(
    name = "processors_test",
    size = "small",
    srcs = ["audit_api_client_test.go"],
    library = ":processors",
    deps = ["//third_party/golang/godebug/pretty"],
)
