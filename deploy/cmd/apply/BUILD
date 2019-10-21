package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

go_binary(
    name = "apply",
    embed = [":go_default_library"],
)

go_library(
    name = "go_default_library",
    srcs = ["apply.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cmd/apply",
    deps = [
        "//apply:go_default_library",
        "//config:go_default_library",
        "//runner:go_default_library",
        "//terraform:go_default_library",
    ],
)

go_test(
    name = "apply_test",
    size = "small",
    srcs = ["apply_test.go"],
    data = [
        "//cmd/apply/testdata",
        "//samples:configs",
        "//samples/spanned_configs:configs",
    ],
    embed = [":go_default_library"],
    # Override default run dir to make it easier to find test files.
    rundir = ".",
    deps = [
        "@com_github_google_cmp//cmp:go_default_library",
    ],
)
