package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "bigquery.go",
        "cloudbuild.go",
        "compute.go",
        "config.go",
        "data_fusion.go",
        "healthcare.go",
        "iam.go",
        "logging.go",
        "monitoring.go",
        "pair.go",
        "project.go",
        "pubsub.go",
        "resource_manager.go",
        "spanner.go",
        "storage.go",
    ],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/config/tfconfig",
    deps = [
        "//runner:go_default_library",
        "@com_github_imdario_mergo//:go_default_library",
    ],
)
