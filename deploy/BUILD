# Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@deploy_deps//:requirements.bzl", "requirement")

exports_files(["project_config.yaml.schema"])

py_binary(
    name = "create_project",
    srcs = ["create_project.py"],
    data = ["//deploy/templates"],
    default_python_version = "PY3",
    deps = [
        requirement("absl-py"),
        requirement("jsonschema"),
        "//deploy/rule_generator",
        "//deploy/utils",
        "//deploy/utils:forseti",
        "//deploy/utils:runner",
    ],
)

py_test(
    name = "create_project_test",
    srcs = ["create_project_test.py"],
    data = ["//deploy/samples:configs"],
    default_python_version = "PY3",
    deps = [
        requirement("absl-py"),
        ":create_project",
        "//deploy/utils:runner",
    ],
)

py_binary(
    name = "grant_forseti_access",
    srcs = ["grant_forseti_access.py"],
    default_python_version = "PY3",
    deps = [
        requirement("absl-py"),
        "//deploy/utils:forseti",
        "//deploy/utils:runner",
    ],
)

py_binary(
    name = "generate_rules",
    srcs = ["generate_rules.py"],
    default_python_version = "PY3",
    deps = [
        requirement("absl-py"),
        "//deploy/rule_generator",
        "//deploy/utils",
    ],
)
