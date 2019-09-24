# Google Cloud Healthcare Data Protection Toolkit

[![GoDoc](https://godoc.org/github.com/GoogleCloudPlatform/healthcare/deploy?status.svg)](https://godoc.org/github.com/GoogleCloudPlatform/healthcare/deploy)

The Data Protection Toolkit (DPT) provides a command-line interface (CLI) to configure 
Google Cloud Platform (GCP) environments.
It wraps existing tools such as Deployment Manager, Terraform, gcloud, and
kubectl to provide an end-to-end deployment solution. While it is developed for
healthcare use cases, the tool can be applied to a wide variety of Google Cloud
use cases.

**Status:** ALPHA

Expect breaking changes. To allow you to migrate easily, deprecated and new 
fields/features for a period of time.

*   [Setup instructions](#setup-instructions)
    *   [Script prerequisites](#script-prerequisites)
    *   [Create groups](#create-groups)
    *   [Create a YAML config](#create-a-yaml-config)
    *   [Create new projects](#create-new-projects)
    *   [Disabled unneeded APIs](#disabled-unneeded-apis)
*   [Updates](#updates)
*   [Resources](#resources)
*   [Debug](#debug)

## Setup instructions

When creating a DPT config, you have the option to have audit logs saved in the
same project as the hosted data (local audit logs) or in a separate project
(remote audit logs). Remote audit logs can be especially beneficial if you have
several data projects and want a centralized place to aggregate them. After creation,
you can't change whether audit logs are saved locally or remotely.

1.  [Complete script prerequisites](#script-prerequisites) the first time you 
    use these scripts.
1.  [Create groups](#create-groups) for the dataset and audit logs (if using
    remote audit logs) project(s).
1.  [Create a YAML config](#create-a-yaml-config) for the project(s) you want to
    deploy.
1.  [Create new projects](#create-new-projects) using the `cmd/apply/apply.go`
    script. This creates the audit logs project (if required) and all data
    hosting projects.

### Script prerequisites

NOTE: If you run the scripts in Google Cloud Shell, the following dependencies are
already available.

-   [Bazel 0.27+](https://docs.bazel.build/versions/master/install.html)

-   [Gcloud SDK](https://cloud.google.com/sdk/install)

-   [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

-   [Go 1.10+](https://golang.org/dl/)

-   [Terraform 0.12](https://www.terraform.io/downloads.html)

### Create groups

Before using the setup scripts, create the following groups for
your dataset projects. The following recommended names are based on best
practices.

*   *Owners Group*: `{PROJECT_ID}-owners@{DOMAIN}`. Provides owners role to the
    project, which allows members to do anything permitted by organization
    policies within the project. Users should only be added to the owners group
    short term.
*   *Auditors Group*: `{PROJECT_ID}-auditors@{DOMAIN}`. The auditors group has
    permission to list resources, view IAM configurations, and view contents of
    audit logs, but not to view any hosted data. If you have multiple data
    projects, you might want a single auditors group across all projects.
*   *Data Read-only Group*: `{PROJECT_ID}-readonly@{DOMAIN}`. Members of this
    group have read-only access to the hosted data in the project.
*   *Data Read/Write Group*: `{PROJECT_ID}-readwrite@{DOMAIN}`. Members of this
    group have permission to read and write hosted data in the project.

If you are using a separate Audit Logs project, then the audit logs project will
need its own Owners Group and an Auditors group, but no data groups.

### Create a YAML config

Edit a copy of the file `samples/project_with_remote_audit_logs.yaml` or
`samples/project_with_local_audit_logs.yaml`, depending on whether you are using
remote or local audit logs. The schema for these YAML files is in
`project_config.yaml.schema`.

*   The `overall` section contains organization and billing details applied to
    all projects. Omit the `organziation_id` if the projects are not being
    deployed under a GCP organization.
*   If using remote audit logs, include the `audit_logs_project` section, which
    describes the project to use to host audit logs.
*   Under `projects`, list each data hosting project to create.

WARNING: these samples use newer configuration that is incompatible with the
previous and require `--enable_new_style_resources` to be passed to the
`cmd/apply/apply.go` script. This flag will soon become default and the old
style configs will be deprecated. Old style config examples can be found
[here](https://github.com/GoogleCloudPlatform/healthcare/tree/5bfa6b72a8077028ead4ff8c498325915180c3b8/deploy/samples).

### Apply project configs

Use the `cmd/apply/apply.go` script to create or update an audit logs project
(if using remote audit logs) and one or more data hosting projects.

1.  Make sure that the user running the script is in the owners group(s) of all
    projects that will be created, including the audit logs project (if used).
    After successful deployment, you should remove the user from these groups.
1.  If not already logged in, run `gcloud init` to log in as a user with
    permission to create projects under the specified organization and billing
    account.
1.  If you provided a `stackdriver_alert_email` in any project, then when
    prompted during the script, follow the instructions to create new
    Stackdriver Accounts for these projects.
1.  Optional: pass a `--projects` flag listing the projects to deploy,
    or omit this flag to deploy all projects.

    NOTE: deploying a project that was previously deployed triggers an
    update.

1.  If the projects are deployed successfully, the script writes a YAML
    with all generated fields in --output_path. These fields are used to
    generate monitoring rules.

```shell
$ git clone https://github.com/GoogleCloudPlatform/healthcare
$ cd healthcare/deploy
# git checkout a commit different from HEAD if necessary.
$ bazel run cmd/apply:apply -- \
  --config_path=${CONFIG_PATH?} \
  --output_path=${OUTPUT_PATH?} \
  --projects=${PROJECTS?}
```

If the script fails at any point, try to correct the error in the input config
file and try again.

## Updates

You might want to modify a project to add additional resources or change an
existing setting. The `cmd/apply/apply.go` script can also be used to update a
project. Listing a previously deployed project in the `--projects` flag (or
omitting the flag for all projects) triggers an update.

WARNING: Deployment Manager runs with deletion policy "ABANDON". Thus, if a
resource is removed from a project, it becomes unmonitored rather than
being deleted. It is the responsibility of the user to manually remove stale
references including IAM bindings and enabled APIs.

## Steps

The script `cmd/apply/apply.go` takes in YAML config files and creates one or
more projects. It creates an audit logs project if `audit_logs_project` is
provided and a forseti project if `forseti` is provided. It then creates a data
hosting project for each project listed under `projects`. For each new project,
the script performs the following steps:

1.  Creates a new GCP project.
1.  Enables billing on the project.
1.  Enables required services:
    *   deploymentmanager.googleapis.com (for Deployment Manager)
    *   cloudresourcemanager.googleapis.com (for project level IAM policy
        changes)
    *   iam.googleapis.com (if custom IAM roles were defined)
1.  Grants the Deployment Manager service account the required roles:
    *   roles/owner
    *   roles/storage.admin
1.  Creates custom boot image(s), if specified on a `gce_instance` resource
1.  Deploys all project resources including:

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

1.  Prompts the user to create a Stackdriver account (this must be
    done using the Stackdriver UI).

1.  Creates Stackdriver Alerts for IAM changes and unexpected Cloud Storage
    bucket access.

1.  If a `forseti` block is defined:

    *   Uses the Forseti Terraform module to deploy a Forseti instance.
    *   Grants permissions for each project to the Forseti service account so
        they may be monitored.

## Rule generation

To generate new rules for the Forseti instance, run the following command:

```shell
$ bazel run cmd/rule_generator:rule_generator -- \
  --config_path=${CONFIG_PATH?} \
  --generated_fields_path=${OUTPUT_PATH?}
```

By default, the rules will be written to the Forseti server bucket.
Alternatively, use `--output_path` to specify a local directory or a different
Cloud Storage bucket to write the rules to.

## Resources

Resources can be added in the config in a uniform way, but could use different
tools underneath to do the actual deployment. Since each resource might have a
very large and complex schema, we cannot cover all of it in our tooling layers.
Thus, we only implement a subset and allow the users to set additional allowed
fields. This is done through the `properties` block available for
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

## Debug

These are solutions to non-typical problems encountered:

#### Log Sink reporting errors on initial deployment

When using the `cmd/apply/apply.go` script, one of the following situations
occurs: 

* the log sink that exports to the logs dataset displays an error
* you receive an email with the subject "[ACTION REQUIRED] Stackdriver
  Logging export config error in {PROJECT_ID}".

Because the log sink was created and given permissions in two separate
deployments, a delay occurred. For more information, see
https://cloud.google.com/logging/docs/export/configure_export_v2#authorization_delays.

If you have successfully deployed the entire project, the error should
disappear on its own. If you continue to see log sink errors, contact Support.

#### Bucket permission denied

During a failed deployment manager deployment, the error message typically 
contains the following line:

```
"ResourceType":"storage.v1.bucket", "ResourceErrorCode":"403"
```

This occurs when the Deployment Manager Service account does not have storage admin
permissions. There can be a delay of up to 7 minutes for permission changes to
propagate.

After recent changes, deployment manager permissions will no longer
be revoked. 

1. Wait at least 7 minutes after changing permissions.
2. If remote audit logs failed to deploy due to this error, re-deploy the audit
   logs project.
3. Retry deployment of all remaining projects that failed. 
