load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

licenses(["notice"])  # Apache 2.0

go_library(
    name = "go_default_library",
    srcs = ["json.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/policygen/terraform",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["json_test.go"],
    data = [
        "//policygen/terraform/testdata",
    ],
    embed = [":go_default_library"],
    # Override default run dir to make it easier to find test files.
    rundir = ".",
    deps = [
        "@com_github_google_cmp//cmp:go_default_library",
        "@com_github_google_cmp//cmp/cmpopts:go_default_library",
    ],
)
