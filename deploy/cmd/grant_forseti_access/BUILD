package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_binary(
    name = "grant_forseti_access",
    embed = [":go_default_library"],
)

go_library(
    name = "go_default_library",
    srcs = ["grant_forseti_access.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cmd/grant_forseti_access",
    deps = [
        "//deploy/apply:go_default_library",
    ],
)
