package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "bigquery_dataset.go",
        "binary_authorization.go",
        "binding.go",
        "chc_dataset.go",
        "config.go",
        "default_resource.go",
        "forseti.go",
        "gce_instance.go",
        "gcs_bucket.go",
        "generated_fields.go",
        "gke_cluster.go",
        "gke_workload.go",
        "iam.go",
        "load.go",
        "logsink.go",
        "metric.go",
        "pair.go",
        "pubsub.go",
        "service_account.go",
    ],
    data = [
        "//deploy:generated_fields.yaml.schema",
        "//deploy:project_config.yaml.schema",
        "//deploy/config/templates",
        "//deploy/templates",
        "//deploy/templates/chc_resource:chc_dataset_files",
    ],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/config",
    deps = [
        "//deploy/config/tfconfig:go_default_library",
        "@com_github_imdario_mergo//:go_default_library",
        "@com_github_mitchellh_homedir//:go_default_library",
        "@com_github_xeipuuv_gojsonschema//:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "bigquery_dataset_test.go",
        "chc_dataset_test.go",
        "default_resource_test.go",
        "forseti_test.go",
        "gce_instance_test.go",
        "gcs_bucket_test.go",
        "generated_fields_test.go",
        "gke_cluster_test.go",
        "iam_test.go",
        "load_test.go",
        "logsink_test.go",
        "metric_test.go",
        "pubsub_test.go",
        "service_account_test.go",
    ],
    data = [
        "//deploy/samples:configs",
    ],
    # Override default run dir to make it easier to find test files.
    rundir = ".",
    deps = [
        ":go_default_library",
        "//deploy/testconf:go_default_library",
        "@com_github_google_cmp//cmp:go_default_library",
        "@com_github_google_cmp//cmp/cmpopts:go_default_library",
        "@in_ghodss_yaml//:go_default_library",
    ],
)
