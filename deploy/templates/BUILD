# GCP Deployment Manager Templates used by the Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@deploy_deps//:requirements.bzl", "requirement")

filegroup(
    name = "templates",
    srcs = [
        "data_project.py",
        "gce_vms.py",
        "remote_audit_logs.py",
    ],
)

# TODO: Change default python version for tests to PY3 once deployment templates support it.

py_library(
    name = "data_project",
    srcs = ["data_project.py"],
)

py_test(
    name = "data_project_test",
    srcs = ["data_project_test.py"],
    python_version = "PY2",
    deps = [
        requirement("absl-py"),
        ":data_project",
    ],
)

py_library(
    name = "gce_vms",
    srcs = ["gce_vms.py"],
)

py_test(
    name = "gce_vms_test",
    srcs = ["gce_vms_test.py"],
    python_version = "PY2",
    deps = [
        requirement("absl-py"),
        ":gce_vms",
    ],
)

py_library(
    name = "remote_audit_logs",
    srcs = ["remote_audit_logs.py"],
)

py_test(
    name = "remote_audit_logs_test",
    srcs = ["remote_audit_logs_test.py"],
    python_version = "PY2",
    deps = [
        requirement("absl-py"),
        ":remote_audit_logs",
    ],
)
