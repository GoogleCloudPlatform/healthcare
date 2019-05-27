# GCP Deployment Manager Templates used by the Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

filegroup(
    name = "templates",
    srcs = [
        "audit_log_config.py",
        "data_project.py",
        "gce_vms.py",
        "remote_audit_logs.py",
    ],
)

py_library(
    name = "data_project",
    srcs = ["data_project.py"],
)

py_test(
    name = "data_project_test",
    srcs = ["data_project_test.py"],
    python_version = "PY3",
    deps = [":data_project"],
)

py_library(
    name = "gce_vms",
    srcs = ["gce_vms.py"],
)

py_test(
    name = "gce_vms_test",
    srcs = ["gce_vms_test.py"],
    python_version = "PY3",
    deps = [":gce_vms"],
)

py_library(
    name = "remote_audit_logs",
    srcs = ["remote_audit_logs.py"],
)

py_test(
    name = "remote_audit_logs_test",
    srcs = ["remote_audit_logs_test.py"],
    python_version = "PY3",
    deps = [":remote_audit_logs"],
)
