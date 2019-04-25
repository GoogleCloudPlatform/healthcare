package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["rule_generator.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cmd/rule_generator",
    visibility = ["//visibility:private"],
    deps = [
        "//deploy/cft:go_default_library",
        "//deploy/rulegen:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
    ],
)

go_binary(
    name = "rule_generator",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
