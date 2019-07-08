package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "apply_forseti",
    embed = [":go_default_library"],
)

go_library(
    name = "go_default_library",
    srcs = ["apply_forseti.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cmd/apply_forseti",
    deps = [
        "//deploy/apply:go_default_library",
        "//deploy/config:go_default_library",
    ],
)
