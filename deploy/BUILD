# Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@bazel_gazelle//:def.bzl", "gazelle")
# gazelle:prefix github.com/GoogleCloudPlatform/healthcare/deploy
gazelle(name = "gazelle")

exports_files(["project_config.yaml.schema"])

py_binary(
    name = "create_project",
    srcs = ["create_project.py"],
    # Pass binary paths as arg due to unstable location in bazel built binaries.
    args = [
        "--apply_binary=$(location //deploy/cmd/apply)",
        "--apply_forseti_binary=$(location //deploy/cmd/apply_forseti)",
        "--rule_generator_binary=$(location //deploy/cmd/rule_generator)",
        "--load_config_binary=$(location //deploy/cmd/load_config)",
        "--grant_forseti_access_binary=$(location //deploy/cmd/grant_forseti_access)",
    ],
    data = [
        "//deploy/cmd/apply",
        "//deploy/cmd/apply_forseti",
        "//deploy/cmd/grant_forseti_access",
        "//deploy/cmd/load_config",
        "//deploy/cmd/rule_generator",
    ],
    python_version = "PY3",
    deps = [":create_project_lib"],
)

py_library(
    name = "create_project_lib",
    srcs = ["create_project.py"],
    data = ["//deploy/templates"],
    deps = [
        "//deploy/utils",
        "//deploy/utils:field_generation",
        "//deploy/utils:forseti",
        "//deploy/utils:runner",
    ],
)

py_test(
    name = "create_project_test",
    srcs = ["create_project_test.py"],
    # Pass binary paths as arg due to unstable location in bazel built binaries.
    args = [
        "--apply_binary=$(location //deploy/cmd/apply)",
        "--apply_forseti_binary=$(location //deploy/cmd/apply_forseti)",
        "--rule_generator_binary=$(location //deploy/cmd/rule_generator)",
        "--load_config_binary=$(location //deploy/cmd/load_config)",
        "--grant_forseti_access_binary=$(location //deploy/cmd/grant_forseti_access)",
    ],
    data = [
        "//deploy/cmd/apply",
        "//deploy/cmd/apply_forseti",
        "//deploy/cmd/grant_forseti_access",
        "//deploy/cmd/load_config",
        "//deploy/cmd/rule_generator",
        "//deploy/samples:configs",
    ],
    python_version = "PY3",
    deps = [
        ":create_project_lib",
        "//deploy/utils",
    ],
)
