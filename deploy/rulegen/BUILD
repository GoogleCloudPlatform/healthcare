package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "bigquery.go",
        "resource.go",
        "rulegen.go",
    ],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/rulegen",
    visibility = ["//visibility:public"],
    deps = [
        "//deploy/cft:go_default_library",
        "@com_github_mitchellh_hashstructure//:go_default_library",
        "@in_gopkg_yaml_v2//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "bigquery_test.go",
        "rulegen_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//deploy/cft:go_default_library",
        "@com_github_google_cmp//cmp:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
        "@in_gopkg_yaml_v2//:go_default_library",
    ],
)
