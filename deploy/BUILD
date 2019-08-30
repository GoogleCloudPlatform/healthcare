# Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@bazel_gazelle//:def.bzl", "gazelle")
# gazelle:prefix github.com/GoogleCloudPlatform/healthcare/deploy
gazelle(name = "gazelle")

exports_files([
    "project_config.yaml.schema",
    "generated_fields.yaml.schema",
])

py_binary(
    name = "create_project",
    srcs = ["create_project.py"],
    # Pass binary paths as arg due to unstable location in bazel built binaries.
    args = [
        "--apply_binary=$(location //cmd/apply)",
    ],
    data = [
        "//cmd/apply",
    ],
    python_version = "PY3",
    deps = [":create_project_lib"],
)

py_library(
    name = "create_project_lib",
    srcs = ["create_project.py"],
    data = ["//templates"],
)
