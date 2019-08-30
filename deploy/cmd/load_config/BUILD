package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "load_config",
    embed = [":go_default_library"],
)

go_library(
    name = "go_default_library",
    srcs = ["load_config.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cmd/load_config",
    deps = [
        "//config:go_default_library",
    ],
)
