load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
git_repository(
    name = "io_bazel_rules_python",
    remote = "https://github.com/bazelbuild/rules_python.git",
    commit = "9835856213b93594859f9c1f789d920264c8ff4f",
)

load("@io_bazel_rules_python//python:pip.bzl", "pip_repositories", "pip_import")
pip_repositories()
pip_import(
    name="deploy_deps",
    requirements="//deploy:requirements.txt",
)

load("@deploy_deps//:requirements.bzl", "pip_install")
pip_install()
