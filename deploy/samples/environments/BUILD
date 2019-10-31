load("//starlark:config_test.bzl", "config_test")

package(default_visibility = ["//visibility:public"])

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
        ],
    )
    for env in [
        "dev",
        "prod",
    ]
]
