load("//starlark:config_test.bzl", "config_test")

package(
    default_applicable_licenses = ["//:license"],
    default_visibility = ["//visibility:public"],
)

licenses(["notice"])

filegroup(
    name = "configs",
    srcs = [
        ":config.yaml",
        ":project.tmpl.yaml",
    ],
)

config_test(
    name = "config_test",
    config = ":config.yaml",
    enable_terraform = True,
    deps = [
        ":project.tmpl.yaml",
        "//samples/full:shared_configs",
    ],
)
