package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

py_binary(
    name = "validate_config",
    srcs = ["validate_config.py"],
    python_version = "PY3",
    deps = [":validate_config_lib"],
)

py_library(
    name = "validate_config_lib",
    srcs = ["validate_config.py"],
)
