load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")
load("@io_bazel_rules_docker//container:container.bzl", "container_image")
load("@io_bazel_rules_docker//go:image.bzl", "go_image")

package(default_visibility = ["//visibility:public"])

licenses(["notice"])  # Apache 2.0

go_binary(
    name = "apply",
    embed = [":go_default_library"],
)

go_library(
    name = "go_default_library",
    srcs = ["apply.go"],
    importpath = "github.com/GoogleCloudPlatform/healthcare/deploy/cmd/apply",
    deps = [
        "//apply:go_default_library",
        "//config:go_default_library",
        "//runner:go_default_library",
        "//terraform:go_default_library",
        "@com_github_google_shlex//:go_default_library",
    ],
)

go_test(
    name = "apply_test",
    size = "small",
    srcs = ["apply_test.go"],
    data = [
        "//cmd/apply/testdata",
        "//samples/full:all_configs",
    ],
    embed = [":go_default_library"],
    # Override default run dir to make it easier to find test files.
    rundir = ".",
    deps = [
        "@com_github_google_cmp//cmp:go_default_library",
    ],
)

container_image(
    name = "base",
    base = "@gcloud//image",
    directory = "/usr/local/bin/",
    files = [
        "@terraform//:executable",
    ],
)

# To build the image:
#   $ bazel build cmd/apply:apply_image.tar
# To run the image:
#   $ docker load -i bazel-bin/cmd/apply/apply_image.tar
#   $ docker run -it --rm \
#       -v ~/.config/gcloud:/root/.config/gcloud \
#       -v /path/to/config/dir:/workdir \
#       bazel/cmd/apply:apply_image \
#       --config_path=/workdir/config.yaml
go_image(
    name = "apply_image",
    base = ":base",
    embed = [":go_default_library"],
)
