package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "apply_resources",
    embed = [":go_default_library"],
)

go_library(
    name = "go_default_library",
    srcs = ["apply_resources.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cmd/apply_resources",
    deps = [
        "//apply:go_default_library",
        "//config:go_default_library",
    ],
)
