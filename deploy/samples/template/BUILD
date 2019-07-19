# Sample deployment configs with templates and tests.

package(default_visibility = ["//visibility:public"])

load("//deploy/starlark:config_test.bzl", "config_test")

licenses(["notice"])  # Apache 2.0

filegroup(
    name = "configs",
    srcs = glob(["*.yaml"]),
)

config_test(
    name = "sample_template_config_test",
    config = ":input.yaml",
    imports = [
        ":overall.tmpl.yaml",
        ":project.tmpl.yaml",
    ],
)
