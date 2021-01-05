load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

go_library(
    name = "go_default_library",
    srcs = [
        "binding.go",
        "config.go",
        "forseti.go",
        "generated_fields.go",
        "load.go",
        "pair.go",
        "terraform.go",
    ],
    data = [
        "//:generated_fields.yaml.schema",
        "//:project_config.yaml.schema",
    ],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/config",
    deps = [
        "//config/tfconfig:go_default_library",
        "@com_github_imdario_mergo//:go_default_library",
        "@com_github_mitchellh_homedir//:go_default_library",
        "@com_github_xeipuuv_gojsonschema//:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "forseti_test.go",
        "generated_fields_test.go",
        "load_test.go",
    ],
    data = [
        "//samples/full:all_configs",
    ],
    # Override default run dir to make it easier to find test files.
    rundir = ".",
    deps = [
        ":go_default_library",
        "//config/tfconfig:go_default_library",
        "//testconf:go_default_library",
        "@com_github_google_cmp//cmp:go_default_library",
        "@com_github_google_cmp//cmp/cmpopts:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
    ],
)
