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

# Healthcare Audit Data Collector.

package(default_visibility = ["//visibility:public"])

load("//apphosting/runtime/go:build_defs.bzl", "go_appengine_binary_zip")

licenses(["notice"])  # Apache 2.0

go_appengine_binary_zip(
    name = "app_zip",
    go_appengine_binary = ":app",
)

# app is the App Engine deployment target
# TODO: replace go_appengine with go_binary to make it work for bazel. Also need to update README to reflect that.
go_appengine_binary(
    name = "app",
    srcs = [],
    configs = ["app.yaml"],
    data = [],
    deps = [
        ":collector",
    ],
)

go_library(
    name = "collector",
    srcs = ["collector.go"],
    deps = [
        "//deploy/collector/processors",
        "//third_party/golang/appengine",
        "//third_party/golang/appengine/log",
        "//third_party/golang/cloud/storage",
        "//third_party/golang/gorilla/mux",
    ],
)
