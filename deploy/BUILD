# Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load("@bazel_gazelle//:def.bzl", "gazelle")

# gazelle:prefix github.com/GoogleCloudPlatform/healthcare/deploy
gazelle(name = "gazelle")

exports_files([
    "project_config.yaml.schema",
    "generated_fields.yaml.schema",
])
