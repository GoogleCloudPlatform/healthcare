package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["deploymentmanager.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/deployment",
    deps = [
        "@in_ghodss_yaml//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["deploymentmanager_test.go"],
    embed = [":go_default_library"],
    deps = [
        "@com_github_google_cmp//cmp:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
    ],
)
