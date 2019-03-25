# Utilities for Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

load("@deploy_deps//:requirements.bzl", "requirement")

py_library(
    name = "forseti",
    srcs = ["forseti.py"],
    deps = [
        requirement("absl-py"),
        ":runner",
    ],
)

py_test(
    name = "forseti_test",
    srcs = ["forseti_test.py"],
    python_version = "PY3",
    deps = [
        requirement("absl-py"),
        requirement("mock"),
        ":forseti",
    ],
)

py_library(
    name = "runner",
    srcs = ["runner.py"],
    deps = [requirement("absl-py")],
)

py_library(
    name = "utils",
    srcs = ["utils.py"],
    data = [
        "//deploy:project_config.yaml.schema",
    ],
    deps = [
        requirement("absl-py"),
        requirement("jsonschema"),
        requirement("ruamel.yaml"),
        ":runner",
    ],
)

py_test(
    name = "utils_test",
    srcs = ["utils_test.py"],
    data = [
        "//deploy/samples:configs",
    ],
    python_version = "PY3",
    deps = [
        requirement("absl-py"),
        ":utils",
    ],
)
