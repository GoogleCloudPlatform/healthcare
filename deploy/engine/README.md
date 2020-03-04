# Terraform Engine

WARNING: This is a pre-alpha version of the future backend of DPT. There will
likely be several iterations before a stable launch.

Terraform Engine is a framework that puts together Terraform resources and
modules into a complete deployment. Users hosting any type of sensitive data on
GCP typically need to go through a few common processes like setting up auditing
and monitoring for a secure environment. By using our out of the box complete
end-to-end examples that implement these steps for you, you can quickly get
to the parts of the infrastructure that drive your business.

This tool will help you follow Terraform
[best practices](https://www.hashicorp.com/resources/evolving-infrastructure-terraform-opencredo),
by defining smaller modular configs rather than monolithic modules that quickly
get out of hand.

Our [templates](./templates) use Google recommended best practice modules from
the [Cloud Foundation Toolkit](https://cloud.google.com/foundation-toolkit).

Our [sample](./samples) configs will show you how to put the templates together
into a complete environment.

## Requirements

- [Terraform 0.12+](https://www.terraform.io/downloads.html)
- [Terragrunt](https://github.com/gruntwork-io/terragrunt/releases)
- [Go](https://golang.org/dl/)
- [Bazel] (https://bazel.build/)

## Usage

```
$ bazel run :main -- --config_path=$PWD/samples/config.yaml --output_dir=/tmp/engine
```

You should now have directories to bootstrap your environment and place your
prod configs.

## Tips

- Each template should be defined so that it is independent or depends on a
  previous template (the one exception being the 'base' template which should
  always be independent and used first).

  This means users can comment out and incrementally uncomment one template and
  deploy. This avoids having to deploy the entire infra all at once.

  Once https://github.com/gruntwork-io/terragrunt/pull/636 is fixed, you
  should also pass `--terragrunt-parallelism=1`.

- Before running `terragrunt apply-all` always run `terragrunt plan-all` and
  carefully review the output.

- `terragrunt apply-all` should be used at least once, or while the org level
  resources are being setup. After that, only subsets should be deployed. This
  reduces the chances of a bad apply affecting multiple parts of your infra.

