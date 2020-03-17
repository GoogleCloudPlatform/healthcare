load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

licenses(["notice"])  # Apache 2.0

go_library(
    name = "go_default_library",
    srcs = ["tfplan.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/policygen/tfplan",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["tfplan_test.go"],
    data = [
        "//policygen/tfplan/testdata",
    ],
    embed = [":go_default_library"],
    # Override default run dir to make it easier to find test files.
    rundir = ".",
)
