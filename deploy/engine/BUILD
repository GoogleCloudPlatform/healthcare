package(default_visibility = ["//visibility:private"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "main",
    embed = [":go_default_library"],
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/engine",
    deps = [
        "//config:go_default_library",
        "//template:go_default_library",
        "@com_github_imdario_mergo//:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
    ],
)
