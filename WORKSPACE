load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "io_bazel_rules_go",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/0.17.7/rules_go-0.17.7.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/0.17.7/rules_go-0.17.7.tar.gz",
    ],
    sha256 = "ce1afed0075e01bd883cfbafbbcc26bfab392eb24854e07a20be78ab13f9e15e",
)
http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
)
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
gazelle_dependencies()

go_repository(
    name = "com_github_google_cmp",
    importpath = "github.com/google/go-cmp",
    tag = "v0.2.0",
)

go_repository(
    name = "com_github_mitchellh_hashstructure",
    importpath = "github.com/mitchellh/hashstructure",
    tag = "v1.0.0",
)

go_repository(
    name = "com_github_mitchellh_homedir",
    importpath = "github.com/mitchellh/go-homedir",
    tag = "v1.1.0",
)

go_repository(
    name = "com_github_imdario_mergo",
    importpath = "github.com/imdario/mergo",
    tag = "v0.3.7",
)

go_repository(
    name = "com_github_xeipuuv_gojsonschema",
    importpath = "github.com/xeipuuv/gojsonschema",
    tag = "v1.1.0",
)

go_repository(
    name = "com_github_xeipuuv_gojsonreference",
    importpath = "github.com/xeipuuv/gojsonreference",
    commit = "bd5ef7bd5415a7ac448318e64f11a24cd21e594b",
)

go_repository(
    name = "com_github_xeipuuv_gojsonpointer",
    importpath = "github.com/xeipuuv/gojsonpointer",
    commit = "4e3ac2762d5f479393488629ee9370b50873b3a6",
)

go_repository(
    name = "in_ghodss_yaml",
    importpath = "github.com/ghodss/yaml",
    commit = "25d852aebe32c875e9c044af3eef9c7dc6bc777f",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    tag = "v2.2.2",
)

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
git_repository(
    name = "io_bazel_rules_python",
    remote = "https://github.com/bazelbuild/rules_python.git",
    commit = "fdbb17a4118a1728d19e638a5291b4c4266ea5b8",
)

http_archive(
    name = "terraform_google_forseti",
    urls = ["https://github.com/forseti-security/terraform-google-forseti/archive/v3.0.0.tar.gz"],
    sha256 = "e9d5d669a395b68d15118d74b96544fca0e5d19aed6aa2362eee4cb9040fba47",
    strip_prefix = "terraform-google-forseti-3.0.0",
    build_file_content = """
filegroup(
  name = "all_files",
  srcs = glob(["**/*"]),
  visibility = ["//visibility:public"],
)""",
)
