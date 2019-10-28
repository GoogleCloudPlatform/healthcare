load("//starlark:config_test.bzl", "config_test")

package(default_visibility = ["//visibility:public"])

licenses(["notice"])

filegroup(
    name = "configs",
    srcs = [":config.yaml"],
)

config_test(
    name = "config_test",
    config = ":config.yaml",
    enable_terraform = True,
    deps = [
        "//samples/full:shared_configs",
    ],
)
