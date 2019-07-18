package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    testonly = 1,
    srcs = [
        "testconf.go",
    ],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/testconf",
    deps = [
        "//deploy/config:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
    ],
)
