load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

# Package tfimport provides functions to import resources from a Terraform config.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

go_library(
    name = "go_default_library",
    srcs = ["tfimport.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/importer/tfimport",
    deps = [
        "//importer/resources:go_default_library",
        "//policygen/terraform:go_default_library",
        "//runner:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["tfimport_test.go"],
    embed = [":go_default_library"],
    # Override default run dir to make it easier to find test files.
    rundir = ".",
    deps = [
        "//importer/resources:go_default_library",
        "//policygen/terraform:go_default_library",
        "@com_github_google_cmp//cmp:go_default_library",
    ],
)
