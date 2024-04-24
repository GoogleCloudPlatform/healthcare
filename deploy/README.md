# Google Cloud Healthcare Data Protection Toolkit

This is the stable version of the Data Protection Toolkit.
It supports creating projects with auditing and monitoring built in. We are
actively working on the next iteration of the tool at
https://github.com/GoogleCloudPlatform/healthcare-data-protection-suite. For new
users, we recommend starting with the new toolkit so you can stay up to date
with the latest features. For current users of this toolkit, they will continue
to function in a stable state

*   [Features](#features)
*   [Use Cases](#use-cases)
*   [Setup Instructions](#setup-instructions)
*   [Phases](#phases)
*   [Rule Generation](#rule-generation)

## Features

DPT provides an end-to-end framework for deploying your entire organizational
structure including central devops, auditing, monitoring and data hosting
projects.

[GCP projects](https://cloud.google.com/docs/enterprise/best-practices-for-enterprise-organizations)
delineate clear boundaries between resources and functions of an organization
and DPT makes it easy to create these projects in a secure and repeatable
manner.

DPT uses existing tools like [Terraform](https://www.terraform.io/) and
[GCloud](https://cloud.google.com/sdk/gcloud/) to work. It can generate multiple
Terraform deployments for a single GCP project and deploy them in the right
order.

DPT can be used to completely manage an organization's projects by creating and
managing all resources within the projects or only handle a subset. For example,
if an organization has a highly custom solution or needs many resources that DPT
does not support natively, they can still use DPT to setup the foundational
pieces such as data projects + centralized auditing, devops and monitoring.
Resources within the project could then be managed through another process,
such as direct use of Terraform.

## Use Cases

- **Macro quickstart**: Package tutorials/quickstarts as deployable templates
  to bring up all necessary projects and resources with one command.

- **Customer onboarding**: Define templates containing all base resources needed
  to get a minimal footprint up and running on GCP quickly.
  Accelerate customer adoption in a security/governance-forward approach.

- **Data analysis/R&D**: Quickly provision multiple identical environments and
  sandboxes for researchers and analysts to use for experimenting on the data
  lake without disrupting production usage. Built in central governance layer
  helps IT and security teams manage their organization's resources.

- **HIPAA/GxP/GDPR alignment**: Build compliance requirements and guidelines
  into templates to ensure that resources are deployed appropriately from the
  beginning. No need to worry about bolting on security monitoring after the
  fact.

- **Repeatable & reproducible deployments**: Define identical
  [environments](samples/environments)
  (e.g. dev, test, and prod) and/or reproducible deployments. This includes
  GxP workloads, medical devices, and other Healthcare scenarios. Use DPT to
  define a template once and deploy it consistently as many times as needed for
  projects handling various Healthcare scenarios.

- **Democratize GCP development**: Share templates across teams and
  organizations, including the open source community. This can then become the
  "easy button" for provisioning and managing GCP resources for common use
  cases.


## Setup Instructions

The officially supported environment for running the toolkit is the GCP Cloud
Shell.

When creating a DPT config, you have the option to have audit and devops
resources saved in the same project as the hosted data (local), or in a central
project (remote). Remote audit and devops resources can be especially beneficial
if you have several data projects and want a centralized place to aggregate
them. This cannot be changed afterwards.

1.  [Complete Script Prerequisites](#script-prerequisites) the first time using
    these scripts.
1.  [Create Groups](#create-groups) for the dataset and audit logs (if using
    remote audit logs) project(s).
1.  [Create a YAML config](#create-a-yaml-config) for the project(s) you want to
    deploy.
1.  [Create New Projects](#create-new-projects) using the `cmd/apply/apply.go`
    script. This will create the audit logs project (if required) and all data
    hosting projects.

### Script Prerequisites

NOTE: If running through Cloud Shell, all of the following dependencies are
already available.

-   [Bazel 0.27+](https://docs.bazel.build/versions/master/install.html)

-   [Gcloud SDK](https://cloud.google.com/sdk/install)

-   [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

-   [Terraform 0.12](https://www.terraform.io/downloads.html)

You will also need to checkout the DPT source code:

```shell
$ git clone https://github.com/GoogleCloudPlatform/healthcare
$ cd healthcare/deploy
# git checkout a commit different from HEAD if necessary.
# All bazel commands will be run from this directory.
```

### Create Groups

Before using the setup scripts, you will need to create the following groups for
your projects. We also provide recommended names based on best
practices.

*   *Owners Group*: `{PROJECT_ID}-owners@{DOMAIN}`. Provides owners role to the
    project, which allows members to do anything permitted by organization
    policies within the project. Users should only be added to the owners group
    short term. Each project should typically have its own owners group.
    When a project needs to be updated, the deployer should only temporarily
    join the owners group of the project being deployed and should not have
    access to other unrelated projects.
*   *Auditors Group*: `{PROJECT_ID}-auditors@{DOMAIN}`. The auditors group has
    permission to list resources, view IAM configurations and view contents of
    audit logs, but not to view any hosted data. If you have multiple data
    projects, you may want a single auditors group across all projects.

#### Supported Resources

DPT natively supports many resources by either using the native
[Terraform Google provider](https://www.terraform.io/docs/providers/google/index.html),
or for more complex resources, such as Forseti, best practices modules from
the [Cloud Foundation Toolkit](https://cloud.google.com/foundation-toolkit/).

For resources using the native Terraform Google provider, DPT will often set
recommended defaults and provide helper fields. These would commonly need a
custom module to be standardized. DPT can instead directly use the native
resource, allowing users to set any field the native resource supports while
still getting the benefits of best practices and standardization.

**Example: Storage Buckets**

Storage buckets are created using the native google provider
[resource](https://www.terraform.io/docs/providers/google/r/storage_bucket.html).

They can be created through DPT by adding a `storage_buckets` field under a
project definition in their DPT config. Any possible bucket field can be set
here.

A user may wish to set custom IAM members on this bucket. DPT provides a helper
field, `_iam_members` to create
[IAM members](https://www.terraform.io/docs/providers/google/r/storage_bucket_iam.html)
on this bucket.

By default, versioning will be enabled and access logs will be exported to the
audit bucket, if set the project `audit.logs_storage_bucket`) field.

### Create a YAML Config

View the schema for the config in `project_config.yaml.schema` and examples in
the `samples` directory for sample YAML configs.

*   The `overall` section contains organization and billing details applied to
    all projects. Omit the `organziation_id` if the projects are not being
    deployed under a GCP organization.
*   If using remote audit logs, include the `audit_logs_project` section, which
    describes the project that will host audit logs.
*   Under `projects`, list each data hosting project to create.


At this point you are ready to deploy it. However, we recommend getting familiar
with what your config will do before proceeding.

```
$ mkdir /tmp/dpt_output
$ bazel run cmd/apply:apply -- \
  --config_path=${CONFIG_PATH?} \
  --projects=${PROJECTS?}
  --dry_run
  --terraform_configs_dir=/tmp/dpt_output
```

You can now investigate the Terraform configs DPT will deploy under
`/tmp/dpt_output` and also see all the commands it will run. It will also run
some basic validation on your configs.

### Apply project configs

Use the `cmd/apply/apply.go` script to create or update your projects.

1.  Make sure the user running the script is in the owners group(s) of all
    projects that will be deployed, including the audit logs project (if used).
    You should remove the user from these groups after successful deployment.
1.  If not already logged in, run `gcloud init` to log in as a user with
    permission to create projects under the specified organization and billing
    account.
1.  If you provided a `monitoring_notification_channels` in any project, then
    when prompted during the script, follow the instructions to create new
    Stackdriver Accounts for these projects.
1.  Optional: pass a `--projects` flag explicitly listing the projects you wish
    to deploy, or omit this flag to deploy all projects.

    NOTE: deploying a project that was previously deployed will trigger an
    update.

1.  If the projects were deployed successfully, the script will write a YAML
    with all generated fields in the file specified in the project config using
    the `generated_fields_path` attribute. These fields are used to generate
    monitoring rules.

```shell
$ bazel run cmd/apply:apply -- \
  --config_path=${CONFIG_PATH?} \
  --projects=${PROJECTS?}
```

If the script fails at any point, try to correct the error in the input config
file and try again.

## Phases

NOTE: This section is only applicable if using `--enable_terraform`.

DPT uses [Terraform](https://www.terraform.io/) as its primary deployment tool
to apply a project's config to GCP.

A project config is broken down to several phases which contain one or more
Terraform deployments. This is used to handle dependencies and make logical
groupings between projects and deployments. Details on each specific field in
the project config can be found in the `project_config.yaml.schema` schema file.

"Base" projects such as central `devops`, `audit` and `forseti` projects go
through each phase as a group to handle dependencies among themselves. Once the
base projects have been deployed, each data hosting project is deployed from
beginning till end.

Additionally, individual deployments in a project can be customized with any
valid [Terraform JSON](https://www.terraform.io/docs/configuration/syntax-json.html)
syntax through the `terraform_deployments` field. See the schema file for more
details on supported deployments.

#### Phase 1: Project

This phase contains a single deployment which creates the project and the bucket
to store this project's remote state defined by the `state_storage_bucket`
field. The state storage bucket will hold the states of deployments related to
this project.

DPT can bootstrap projects and remote state buckets to avoid the
[chicken and egg problem](https://www.google.com/search?q=terraform+remote+state+chicken+and+egg+problem&oq=terraform+remote+state+chicken+and+egg+problem)
which usually need a separate solution.

NOTE: if a central `devops` field is set in the config, all state buckets will
be created in the devops project.

#### Phase 2: Resources

This phase contains several deployment which create both user defined and
default resources in the project.

**services**: This deployment contains default set of services based on the
resources defined in the project config as well as the user defined services set
by the `project_services` field in the project config.

**resources**: This deployment contains deployment contains some preset
defaults, such as default IAM permissions (to grant the `owners_group` owners
access to the project), logging metrics and alert policies.

It also contains user defined resources that can be set by the following fields
in the project config:

-   `bigquery_datasets`
-   `compute_firewalls`
-   `compute_images`
-   `compute_instances`
-   `healthcare_datasets`
-   `monitoring_notification_channels`
-   `project_iam_custom_roles`
-   `project_iam_members`
-   `pubsub_topics`
-   `resource_manager_liens`
-   `service_accounts`
-   `storage_buckets`

Following this, the user who created the project is removed as an owner, thus
going forward only those in the owners group can make changes to the project.

#### Phase 3: Audit

This phase contains a single deployment which creates audit log resources
defined in the `audit` block of a project (BigQuery Dataset and Cloud Storage
Bucket) as well as logging sinks to export audit logs.

NOTE: If a top-level `audit` block is set in the config, these resources will be
created in the central audit project.

#### Phase 4: Forseti

NOTE: This phase is only applicable if a `forseti` block is set in the config.

**forseti**: If the Forseti project config is also being applied, a forseti
instance is applied in the Forseti project at this point.

**forseti-permissions**: The Forseti instance is granted the minimum necessary
access to each project to monitor security violations in them.
