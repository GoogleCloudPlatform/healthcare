# Internal utilities for use with the CHC Forseti instance.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

py_library(
    name = "project_config",
    srcs = ["project_config.py"],
)

py_test(
    name = "project_config_test",
    size = "small",
    srcs = ["project_config_test.py"],
    deps = [":project_config"],
)

py_binary(
    name = "rule_generator",
    srcs = ["rule_generator.py"],
    deps = [
        ":project_config",
        "//deploy/rule_generator/scanners:audit_logging_scanner_rules",
        "//deploy/rule_generator/scanners:bigquery_scanner_rules",
        "//deploy/rule_generator/scanners:bucket_scanner_rules",
        "//deploy/rule_generator/scanners:cloudsql_scanner_rules",
        "//deploy/rule_generator/scanners:enabled_apis_scanner_rules",
        "//deploy/rule_generator/scanners:iam_scanner_rules",
        "//deploy/rule_generator/scanners:lien_scanner_rules",
        "//deploy/rule_generator/scanners:location_scanner_rules",
        "//deploy/rule_generator/scanners:log_sink_scanner_rules",
    ],
)
