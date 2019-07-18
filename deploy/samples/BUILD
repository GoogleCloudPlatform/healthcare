# Sample deployment configs and tests.

package(default_visibility = ["//visibility:public"])

load("//deploy/starlark:config_test.bzl", "config_test")

licenses(["notice"])  # Apache 2.0

filegroup(
    name = "configs",
    srcs = glob(["**/*.yaml"]),
)

config_test(
    name = "test_sample_minimal",
    config = "//deploy/samples:minimal.yaml",
)

config_test(
    name = "test_sample_project_with_local_audit_logs",
    config = "//deploy/samples:project_with_local_audit_logs.yaml",
)

config_test(
    name = "test_sample_project_with_remote_audit_logs",
    config = "//deploy/samples:project_with_remote_audit_logs.yaml",
)
