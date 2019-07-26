# Utilities for Cloud Healthcare deployment tools.

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

py_library(
    name = "forseti",
    srcs = ["forseti.py"],
    deps = [":runner"],
)

py_library(
    name = "runner",
    srcs = ["runner.py"],
)

py_library(
    name = "utils",
    srcs = ["utils.py"],
    data = [
        "//deploy:project_config.yaml.schema",
    ],
    deps = [":runner"],
)

py_library(
    name = "field_generation",
    srcs = ["field_generation.py"],
    deps = [":utils"],
)

py_test(
    name = "field_generation_test",
    srcs = ["field_generation_test.py"],
    python_version = "PY3",
    deps = [":field_generation"],
)
