package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "bigquery_dataset.go",
        "cft.go",
        "default_resource.go",
        "deployment.go",
        "gke_workload.go",
        "resourcepair.go",
    ],
    data = ["//deploy/cft/templates"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cft",
    deps = [
        "@com_github_imdario_mergo//:go_default_library",
        "@in_gopkg_yaml//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "bigquery_dataset_test.go",
        "cft_test.go",
        "default_resource.go",
        "deployment_test.go",
        "gke_workload_test.go",
        "resourcepair_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "@com_github_google_cmp//cmp:go_default_library",
        "@in_gopkg_yaml//:go_default_library",
    ],
)
