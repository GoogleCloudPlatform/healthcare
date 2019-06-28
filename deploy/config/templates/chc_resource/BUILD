# GCP Deployment Manager Templates used by the Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])

py_library(
    name = "chc_dataset",
    srcs = ["chc_dataset.py"],
)

py_test(
    name = "chc_dataset_test",
    srcs = ["chc_dataset_test.py"],
    python_version = "PY3",
    deps = [":chc_dataset"],
)

filegroup(
    name = "chc_dataset_files",
    srcs = glob(["*.py"]) + glob(["*.schema"]) + glob(["*.yaml"]),
)
