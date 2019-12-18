load("//starlark:config_test.bzl", "config_test")

package(default_visibility = ["//visibility:public"])

licenses(["notice"])

config_test(
    name = "sample_minimal_config_test",
    config = ":minimal.yaml",
    deps = [":generated_fields.yaml"],
)

config_test(
    name = "sample_project_with_local_audit_logs_config_test",
    config = ":project_with_local_audit_logs.yaml",
    deps = [
        ":generated_fields.yaml",
        "@terraform_google_forseti//:all_files",
    ],
)

config_test(
    name = "sample_project_with_remote_audit_logs_config_test",
    config = ":project_with_remote_audit_logs.yaml",
    deps = [
        ":generated_fields.yaml",
        "@terraform_google_forseti//:all_files",
    ],
)
