load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

# Importer imports resources from existing Terraform configs.

licenses(["notice"])  # Apache 2.0

go_binary(
    name = "importer",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/importer",
    visibility = ["//visibility:private"],
    deps = [
        "//config:go_default_library",
        "//importer/tfimport:go_default_library",
        "//policygen/terraform:go_default_library",
        "//runner:go_default_library",
    ],
)
