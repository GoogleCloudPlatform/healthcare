# Sample deployment configs with templates and tests.

package(default_visibility = ["//visibility:public"])

load("//starlark:config_test.bzl", "config_test")

licenses(["notice"])  # Apache 2.0

filegroup(
    name = "configs",
    srcs = glob([
        "*.yaml",
        "base/*.yaml",
        "projects/*.yaml",
    ]),
)

config_test(
    name = "sample_spanned_configs_test",
    config = ":root.yaml",
    deps = [
        "base/base.yaml",
        "projects/partial_audit.yaml",
        "projects/partial_other_projects.yaml",
        "projects/partial_projects.yaml",
        ":generated_fields.yaml",
    ],
)
