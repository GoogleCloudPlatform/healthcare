load("//starlark:config_test.bzl", "config_test")

package(
    default_applicable_licenses = ["//:license"],
    default_visibility = ["//visibility:public"],
)

licenses(["notice"])

[
    config_test(
        name = "%s_config_test" % env,
        config = ":%s.yaml" % env,
        enable_terraform = True,
        deps = [
            ":generated_fields.yaml",
            ":overall.yaml",
            ":project.tmpl.yaml",
            "@terraform_google_forseti//:all_files",
        ],
    )
    for env in [
        "dev",
        "prod",
    ]
]
