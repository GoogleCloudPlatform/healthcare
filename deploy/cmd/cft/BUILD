package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "cft",
    embed = [":go_default_library"],
)

go_library(
    name = "go_default_library",
    srcs = ["cft.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cmd/cft",
    deps = [
        "//deploy/cft:go_default_library",
        "@in_gopkg_yaml//:go_default_library",
    ],
)
