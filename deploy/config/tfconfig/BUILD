package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "bigquery_dataset.go",
        "config.go",
        "pair.go",
        "storage_bucket.go",
    ],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig",
    deps = [
        "@com_github_imdario_mergo//:go_default_library",
    ],
)
