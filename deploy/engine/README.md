# Terraform Engine

Status: ALPHA

Terraform Engine is a framework to jump start your organization onto GCP.
It is a Terraform module generator that puts together Terraform resources and
modules into deployments complete with values for your infrastructure.

## Why?

Users hosting any type of sensitive data on GCP typically need to go through a
few common and repetitive processes such as setting up devops, auditing and
monitoring. By using our out of the box complete end-to-end configs that
implement these steps for you, you can quickly setup a secure and compliant
environment and focus on the parts of the infrastructure that drive your
business.

This tool will help you follow Terraform
[best practices](https://www.hashicorp.com/resources/evolving-infrastructure-terraform-opencredo),
by using the popular open source tool
[Terragrunt](https://terragrunt.gruntwork.io/)
to define smaller modular configs rather than monolithic modules that quickly
get out of hand or need a custom pipeline to manage.

Our templates use Google's best practice modules from the
[Cloud Foundation Toolkit](https://cloud.google.com/foundation-toolkit).

Use our [sample](./samples) configs to quickly get started.

## Requirements

- [Terraform 0.12+](https://www.terraform.io/downloads.html)
- [Terragrunt](https://github.com/gruntwork-io/terragrunt/releases)
- [Go](https://golang.org/dl/)
- [Bazel](https://bazel.build/)

## Usage

The engine takes a path to an input config and a path to output the generated
Terraform configs. After the output has been generated, there is no dependency
on the engine any longer, and the user can directly use the `terraform` and
`terragrunt` binaries to deploy the infrastructure.

Replace the values in [samples/config.yaml](./samples/config.yaml) with values
for your infrastructure, then run the following commands:

```
# Generate Terraform configs.
$ OUTPUT_DIR=/tmp/engine
$ bazel run :main -- --config_path=$PWD/samples/config.yaml --output_path=$OUTPUT_DIR

# Run one time bootstrap to setup devops project to host Terraform state.
$ cd $OUTPUT_DIR/bootstrap
$ terraform init
$ terraform plan
$ terraform apply

# Optional: Backup bootstrap state.
$ nano main.tf
# Uncomment GCS backend block...
$ terraform init

# Deploy org infrastructure.
$ cd $OUTPUT_DIR/org
$ terragrunt init-all
$ terragrunt plan-all
$ terragrunt apply-all

# Optional: Modify and/or add deployments as needed...
$ cd $OUTPUT_DIR/org/.../example-deployment
$ terragrunt init
$ terragrunt plan
$ terragrunt apply
```

## Tips

- Before running `terragrunt apply-all` always run `terragrunt plan-all` and
  carefully review the output. Look for the values of the known fields to ensure
  they are what you expect. You may see some values with the word "mock" in
  them. These values are coming from other deployments and will be filled with
  the real value once Terragrunt runs the dependent deployment.

- `terragrunt apply-all` should be used at least once to deploy your baseline
  org. After that, individual deployments should only be deployed one at a time
  using `terragrunt apply`. This reduces the chances of a bad apply affecting
  multiple parts of your infra and reduces the permissions needed.

