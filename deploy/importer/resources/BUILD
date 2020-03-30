load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

# Package resources defines resource-specific implementations for interface Importer.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

go_library(
    name = "go_default_library",
    srcs = [
        "resources.go",
        "storage.go",
    ],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/importer/resources",
    deps = ["//policygen/terraform:go_default_library"],
)

go_test(
    name = "go_default_test",
    srcs = ["resources_test.go"],
    embed = [":go_default_library"],
    # Override default run dir to make it easier to find test files.
    rundir = ".",
)
