# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.9.0/rules_go-0.9.0.tar.gz",
    sha256 = "4d8d6244320dd751590f9100cf39fd7a4b75cd901e1f3ffdfd6f048328883695",
)

http_archive(
    name = "bazel_gazelle",
    url = "https://github.com/bazelbuild/bazel-gazelle/releases/download/0.9/bazel-gazelle-0.9.tar.gz",
    sha256 = "0103991d994db55b3b5d7b06336f8ae355739635e0c2379dea16b8213ea5a223",
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains", "go_repository")

go_repository(
    name = "io_opencensus_go",
    commit = "e3cae8bbf0ad88a4fa08b3989da469d9e8d7792e",
    importpath = "go.opencensus.io",
)

go_repository(
    name = "org_golang_google_grpc",
    commit = "583a6303969ea5075e9bd1dc4b75805dfe66989a",
    importpath = "google.golang.org/grpc",
)

go_repository(
    name = "org_golang_google_api",
    commit = "ab90adb3efa287b869ecb698db42f923cc734972",
    importpath = "google.golang.org/api",
)

go_repository(
    name = "com_github_golang_protobuf",
    commit = "bbd03ef6da3a115852eaf24c8a1c46aeb39aa175",
    importpath = "github.com/golang/protobuf",
)

go_repository(
    name = "org_golang_google_genproto",
    commit = "2b5a72b8730b0b16380010cfe5286c42108d88e7",
    importpath = "google.golang.org/genproto",
)

go_repository(
    name = "com_github_googleapis_gax_go",
    commit = "317e0006254c44a0ac427cc52a0e083ff0b9622f",
    importpath = "github.com/googleapis/gax-go",
)

go_repository(
    name = "com_google_cloud_go",
    commit = "dfdc5040dbf96e12b7ad6d8080d0c8bd380cb8f2",
    importpath = "cloud.google.com/go",
)

go_repository(
    name = "org_golang_x_oauth2",
    commit = "543e37812f10c46c622c9575afd7ad22f22a12ba",
    importpath = "golang.org/x/oauth2",
)

go_repository(
    name = "org_golang_x_sync",
    commit = "fd80eb99c8f653c847d294a001bdf2a3a6f768f5",
    importpath = "golang.org/x/sync",
)

go_repository(
    name = "org_golang_x_net",
    commit = "cbe0f9307d0156177f9dd5dc85da1a31abc5f2fb",
    importpath = "golang.org/x/net",
)

go_repository(
    name = "com_github_kylelemons_godebug",
    commit = "d65d576e9348f5982d7f6d83682b694e731a45c6",
    importpath = "github.com/kylelemons/godebug",
)

go_rules_dependencies()

go_register_toolchains()

git_repository(
    name = "io_bazel_rules_docker",
    commit = "198367210c55fba5dded22274adde1a289801dc4",
    remote = "https://github.com/bazelbuild/rules_docker.git",
)

load("@io_bazel_rules_docker//go:image.bzl", _go_image_repos = "repositories")

_go_image_repos()

load("@io_bazel_rules_docker//container:container.bzl", container_repositories = "repositories", "container_pull")

container_repositories()

container_pull(
    name = "ubuntu",
    registry = "gcr.io",
    repository = "cloud-marketplace/google/ubuntu16_04",
    digest = "sha256:c81e8f6bcbab8818fdbe2df6d367990ab55d85b4dab300931a53ba5d082f4296",
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()
