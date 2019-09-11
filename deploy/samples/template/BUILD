# Sample deployment configs with templates and tests.

package(default_visibility = ["//visibility:public"])

load("//starlark:config_test.bzl", "config_test")

licenses(["notice"])  # Apache 2.0

filegroup(
    name = "configs",
    srcs = glob(["*.yaml"]),
)

config_test(
    name = "sample_template_config_test",
    config = ":input.yaml",
    deps = [
        ":overall.tmpl.yaml",
        ":project.tmpl.yaml",
        "//samples:generated_fields.yaml",
    ],
)
