load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

licenses(["notice"])  # Apache 2.0

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/policygen",
    visibility = ["//visibility:private"],
    deps = [
        "//config:go_default_library",
        "//policygen/terraform:go_default_library",
        "//runner:go_default_library",
    ],
)

go_binary(
    name = "policygen",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
