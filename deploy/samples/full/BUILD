load("//starlark:config_test.bzl", "config_test")

package(
    default_applicable_licenses = ["//:license"],
    default_visibility = ["//visibility:public"],
)

licenses(["notice"])

filegroup(
    name = "shared_configs",
    srcs = [
        ":generated_fields.yaml",
        ":shared.yaml",
        "@terraform_google_forseti//:all_files",
    ],
)

# Used by tests.
filegroup(
    name = "all_configs",
    srcs = [
        ":shared_configs",
        "//samples/full/team1:configs",
        "//samples/full/team2:configs",
    ],
)

config_test(
    name = "shared_config_test",
    config = ":shared.yaml",
    enable_terraform = True,
    deps = [
        ":generated_fields.yaml",
        "@terraform_google_forseti//:all_files",
    ],
)
