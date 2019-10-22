# Google Cloud Healthcare Data Protection Toolkit

[![GoDoc](https://godoc.org/github.com/GoogleCloudPlatform/healthcare/deploy?status.svg)](https://godoc.org/github.com/GoogleCloudPlatform/healthcare/deploy)

The Data Protection Toolkit (DPT) provides a CLI to configure GCP environments.
It wraps existing tools such as Deployment Manager, Terraform, Gcloud and
Kubectl to provide an end to end deployment solution. While it is developed for
healthcare use cases, the tool can be applied to a wide variety of users wishing
to use Google Cloud.

**Status:** ALPHA

Expect breaking changes. We ensure that we support both deprecated and new
fields/features for a period of time to allow users to migrate easily.

*   [Setup Instructions](#setup-instructions)
*   [Phases](#phases)
*   [Rule Generation](#rule-generation)
*   [Legacy](#legacy)

## Setup Instructions

When creating a DPT config, you have the option to have audit logs saved in the
same project as the hosted data (local audit logs), or in a separate project
(remote audit logs). Remote audit logs can be especially beneficial if you have
several data projects and want a centralized place to aggregate them. This
cannot be changed afterwards.

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

-   [Go 1.10+](https://golang.org/dl/)

-   [Terraform 0.12](https://www.terraform.io/downloads.html)

### Create Groups

Before using the setup scripts, you will need to create the following groups for
your dataset projects. We also provide recommended names based on best
practices.

*   *Owners Group*: `{PROJECT_ID}-owners@{DOMAIN}`. Provides owners role to the
    project, which allows members to do anything permitted by organization
    policies within the project. Users should only be added to the owners group
    short term.
*   *Auditors Group*: `{PROJECT_ID}-auditors@{DOMAIN}`. The auditors group has
    permission to list resources, view IAM configurations and view contents of
    audit logs, but not to view any hosted data. If you have multiple data
    projects, you may want a single auditors group across all projects.
*   *Data Read-only Group*: `{PROJECT_ID}-readonly@{DOMAIN}`. Members of this
    group have read-only access to the hosted data in the project.
*   *Data Read/Write Group*: `{PROJECT_ID}-readwrite@{DOMAIN}`. Members of this
    group have permission to read and write hosted data in the project.

If you are using a separate Audit Logs project, then the audit logs project will
also need its own Owners Group and an Auditors group, but no data groups.

### Create a YAML Config

Edit a copy of the file `samples/project_with_remote_audit_logs.yaml` or
`samples/project_with_local_audit_logs.yaml`, depending on whether you are using
remote or local audit logs. The schema for these YAML files is in
`project_config.yaml.schema`.

*   The `overall` section contains organization and billing details applied to
    all projects. Omit the `organziation_id` if the projects are not being
    deployed under a GCP organization.
*   If using remote audit logs, include the `audit_logs_project` section, which
    describes the project that will host audit logs.
*   Under `projects`, list each data hosting project to create.

WARNING: these samples use newer configuration that is incompatible with the
previous and require `--enable_new_style_resources` to be passed to the
`cmd/apply/apply.go` script. This flag will soon become default and the old
style configs will be deprecated. Old style config examples can be found
[here](https://github.com/GoogleCloudPlatform/healthcare/tree/5bfa6b72a8077028ead4ff8c498325915180c3b8/deploy/samples).

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
$ git clone https://github.com/GoogleCloudPlatform/healthcare
$ cd healthcare/deploy
# git checkout a commit different from HEAD if necessary.
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

## Rule Generation

NOTE: this is currently not supported for Terraform configs.

To generate new rules specific to your config for the Forseti instance, run the
following command:

```shell
$ bazel run cmd/rule_generator:rule_generator -- \
  --config_path=${CONFIG_PATH?}
```

By default, the rules will be written to the Forseti server bucket.
Alternatively, use `--output_path` to specify a local directory or a different
Cloud Storage bucket to write the rules to.


## Legacy

### Steps

The script `cmd/apply/apply.go` takes in YAML config files and creates one or
more projects. It creates an audit logs project if `audit_logs_project` is
provided and a forseti project if `forseti` is provided. It then creates a data
hosting project for each project listed under `projects`. For each new project,
the script performs the following steps:

1.  Create a new GCP project.
1.  Enable billing on the project.
1.  Enable required services:
    *   deploymentmanager.googleapis.com (for Deployment Manager)
    *   cloudresourcemanager.googleapis.com (for project level IAM policy
        changes)
    *   iam.googleapis.com (if custom IAM roles were defined)
1.  Grant the Deployment Manager service account the required roles:
    *   roles/owner
    *   roles/storage.admin
1.  Create custom boot image(s), if specified on a `gce_instance` resource
1.  Deploy all project resources including:

    *   Default resources:
        *   IAM Policies to grant owner and auditor groups project level access
        *   Log sink export to the `logs_bq_dataset`
        *   Log metrics for alerting on bigquery ACL, IAM and bucket ALC
            accesses.
    *   User defined resources (possible resources can be found in the
        `project_config.yaml.schema` file under the `gcp_project.resources`
        definition).

    TIP: To view deployed resources, open the project in the GCP console and
    navigate to the Deployment Manager page. Click the expanded config under the
    `data-protect-toolkit-resources` deployment to view the list of resources
    deployed.

1.  Deploys audit resources to hold exported logs

    *   The audit resources will be deployed in the remote audit logs project if
        one was specified.

1.  Removes the user from the project's owners.

1.  Prompts the user to create a Stackdriver account (currently this must be
    done using the Stackdriver UI).

1.  Creates Stackdriver Alerts for IAM changes and unexpected Cloud Storage
    bucket access.

1.  If a `forseti` block is defined:

    *   Uses the Forseti Terraform module to deploy a Forseti instance.
    *   Grants permissions for each project to the Forseti service account so
        they may be monitored.

### Resources
This section is for the legacy Deployment Manager support. This is enabled
by passing --enable_terraform=false. Deployment manager support will be shut
down in the near future.

Resources can be added in the config in a uniform way, but may use different
tools underneath to do the actual deployment. Since each resource may have a
very large and complex schema, we cannot cover all of it in our tooling layers.
Thus, we only implement a subset and allow the users to set additional allowed
fields as they wish. This is done through the `properties` block available for
most resources. See documentation below for each resource.

See the `samples/` directory for sample resource definitions.

NOTE: project_config.yaml.schema provides documentation on our subset. Please
reference it before adding your resource to your config.

NOTE: Dependencies can be set implicitly in deployment manager resource
properties. See
https://cloud.google.com/deployment-manager/docs/step-by-step-guide/using-references.
Dependencies are only supported between deployment manager resources.

Resource                | Deployment Tool
----------------------- | ---------------
bq_dataset              | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/bigquery)
chc_datasets (ALPHA)    | [Deployment Manager](https://cloud.google.com/healthcare/docs/)
enabled_api             | [Gcloud](https://cloud.google.com/sdk/gcloud/reference/services/enable)
forseti                 | [Terraform (CFT)](https://github.com/forseti-security/terraform-google-forseti)
gce_firewall            | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/firewall)
gce_instance            | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/instance)
gcs_bucket              | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/gcs_bucket)
gke_cluster             | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/gke)
gke_workload            | [Kubectl](https://kubernetes.io/docs/tutorials/configuration)
iam_custom_role         | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/iam_custom_role)
iam_policy              | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/iam_member)
pubsub                  | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/pubsub)
vpc_network             | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/network)
service_account (ALPHA) | [Deployment Manager (Direct)](https://cloud.google.com/iam/reference/rest/v1/projects.serviceAccounts/create)

### Updates

You may wish to modify a project to add additional resources or change an
existing setting. The `cmd/apply/apply.go` script can be used to also update a
project. Listing a previously deployed project in the `--projects` flag (or
omitting the flag for all projects), will trigger an update.

WARNING: Deployment Manager will run with deletion policy "ABANDON". Thus, if a
resource is removed from a project, it will become unmonitored rather than
deleted. It is the responsibility of the user to manually remove stale
references including IAM bindings and enabled APIs.


### Debug

These are solutions to non-typical problems encountered:

#### Log Sink reporting errors on initial deployment

When using the `cmd/apply/apply.go` script, you may see an error on the log sink
that exports to the logs dataset or you may have gotten an email with the
subject "[ACTION REQUIRED] Stackdriver Logging export config error in
{PROJECT_ID}".

This is due to the log sink being created and given permissions in two separate
deployments, causing an inevitable delay.

See
https://cloud.google.com/logging/docs/export/configure_export_v2#authorization_delays.

If you have successfully deployed the entire project then the error should
disappear on its own. If you continue to see log sink errors please reach out
for support.

#### Bucket Permission Denied

Typically the error message will contain the following during a failed
deployment manager deployment:

```
"ResourceType":"storage.v1.bucket", "ResourceErrorCode":"403"
```

This is due to the Deployment Manager Service account not having storage admin
permissions. There can be a delay of up to 7 minutes for permission changes to
propagate. After recent changes, deployment manager permissions will no longer
be revoked so just retry deployment of all projects that failed after at least 7
minutes.

NOTE: if remote audit logs failed to deploy due to this error then you will need
to re-deploy the audit logs project first.
