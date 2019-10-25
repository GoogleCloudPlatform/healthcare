load("//starlark:config_test.bzl", "config_test")

package(default_visibility = ["//visibility:public"])

licenses(["notice"])

filegroup(
    name = "shared_configs",
    srcs = [
        ":generated_fields.yaml",
        ":shared.yaml",
    ],
)

config_test(
    name = "shared_config_test",
    config = ":shared.yaml",
    enable_terraform = True,
    deps = [":generated_fields.yaml"],
)
