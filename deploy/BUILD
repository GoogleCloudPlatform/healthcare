# Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@deploy_deps//:requirements.bzl", "requirement")

exports_files(["project_config.yaml.schema"])

py_binary(
    name = "create_project",
    srcs = ["create_project.py"],
    deps = [
        requirement("absl-py"),
        requirement("jsonschema"),
        "//deploy/utils",
        "//deploy/utils:forseti",
        "//deploy/utils:runner",
    ],
)

py_test(
    name = "create_project_test",
    srcs = ["create_project_test.py"],
    deps = [
        requirement("absl-py"),
        ":create_project",
    ],
)

py_binary(
    name = "grant_forseti_access",
    srcs = ["grant_forseti_access.py"],
    deps = [
        requirement("absl-py"),
        "//deploy/utils:forseti",
        "//deploy/utils:runner",
    ],
)
