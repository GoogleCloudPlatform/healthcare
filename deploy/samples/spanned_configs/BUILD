# Sample deployment configs with templates and tests.

package(default_visibility = ["//visibility:public"])

load("//starlark:config_test.bzl", "config_test")

licenses(["notice"])  # Apache 2.0

filegroup(
    name = "configs",
    srcs = glob(["*.yaml"]),
)

config_test(
    name = "sample_spanned_configs_test",
    config = ":root.yaml",
    imports = [
        ":partial_audit.yaml",
        ":partial_other_projects.yaml",
        ":partial_projects.yaml",
    ],
)
