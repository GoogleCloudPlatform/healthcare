# Sample deployment configs and tests.

package(default_visibility = ["//visibility:public"])

load("//starlark:config_test.bzl", "config_test")

licenses(["notice"])  # Apache 2.0

filegroup(
    name = "configs",
    srcs = glob(["**/*.yaml"]) + [
        "//samples/template:configs",
    ],
)

config_test(
    name = "sample_minimal_config_test",
    config = "//samples:minimal.yaml",
)

config_test(
    name = "sample_project_with_local_audit_logs_config_test",
    config = "//samples:project_with_local_audit_logs.yaml",
)

config_test(
    name = "sample_project_with_remote_audit_logs_config_test",
    config = "//samples:project_with_remote_audit_logs.yaml",
)
