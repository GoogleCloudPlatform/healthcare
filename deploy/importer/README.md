# Terraform Importer

**WARNING: This is an experimental version which does not yet support all resources that DPT does.**

Terraform Importer is a standalone utility to import existing infrastructure into a Terraform state, based on existing Terraform configs. Its target use-case is to facilitate migration between other infrastructure-as-code tools (such as [Deployment Manager](https://cloud.google.com/deployment-manager)), and a planned future version of DPT that will generate Terraform configs.

## Requirements

- [Terraform 0.12+](https://www.terraform.io/downloads.html)
- [Go](https://golang.org/dl/)
- [Bazel](https://bazel.build/)

## Usage

```
$ bazel run //importer -- --input_dir ~/path/to/terraform/configs/dir/
```

On a successful run, the Terraform configs directory will be initialized and all supported resources will be in the state.

If you now run `terraform plan`, you should only see planned changes for differences between the configs and the actual infrastructure.

## Supported Resources

* [google_storage_bucket](https://www.terraform.io/docs/providers/google/r/storage_bucket.html)

## Terraformer

This tool is complementary to [Terraformer](https://github.com/GoogleCloudPlatform/terraformer).

Terraformer focuses on generating new Terraform configs from existing infrastructure, while the Importer allows you to define and organize your own Terraform configs, including importing resources from within modules.
