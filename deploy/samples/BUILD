# Sample deployment configs and tests.
load("//starlark:config_test.bzl", "config_test")

package(default_visibility = ["//visibility:public"])

licenses(["notice"])

filegroup(
    name = "configs",
    srcs = glob(["**/*.yaml"]) + [
        "//samples/template:configs",
    ],
)

config_test(
    name = "sample_minimal_config_test",
    config = "//samples:minimal.yaml",
    deps = ["//samples:generated_fields.yaml"],
)

config_test(
    name = "sample_project_with_local_audit_logs_config_test",
    config = "//samples:project_with_local_audit_logs.yaml",
    deps = ["//samples:generated_fields.yaml"],
)

config_test(
    name = "sample_project_with_remote_audit_logs_config_test",
    config = "//samples:project_with_remote_audit_logs.yaml",
    deps = ["//samples/spanned_configs:generated_fields.yaml"],
)

config_test(
    name = "sample_terraform_config_test",
    config = "//samples:terraform.yaml",
    enable_terraform = True,
    deps = ["//samples/spanned_configs:generated_fields.yaml"],
)
