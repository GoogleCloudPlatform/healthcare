go_library(
    name = "go_default_library",
    srcs = [
        "funcmap.go",
        "template.go",
    ],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/template",
    visibility = ["//visibility:public"],
    deps = ["@com_github_imdario_mergo//:go_default_library"],
)

go_test(
    name = "go_default_test",
    srcs = ["template_test.go"],
    embed = [":go_default_library"],
    deps = ["@com_github_google_cmp//cmp:go_default_library"],
)
