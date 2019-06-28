# Google Cloud Healthcare Data Protection Toolkit

[![GoDoc](https://godoc.org/github.com/GoogleCloudPlatform/healthcare/deploy?status.svg)](https://godoc.org/github.com/GoogleCloudPlatform/healthcare/deploy)

Tools to configure GCP environments aimed at healthcare use cases.

*   [Setup Instructions](#setup-instructions)
    *   [Script Prerequisites](#script-prerequisites)
    *   [Create Groups](#create-groups)
    *   [Create a YAML config](#create-a-yaml-config)
    *   [Create New Projects](#create-new-projects)
    *   [Disabled Unneeded APIs](#disabled-unneeded-apis)
*   [Updates](#updates)
*   [Deployment Manager Templates](#deployment-manager-templates)
    *   [Template data_project.py](#template-data_projectpy)
    *   [Template remote_audit_logs.py](#template-remote_audit_logspy)
    *   [Script create_project.py](#script-create_projectpy)
*   [Resources](#resources)
*   [Debug](#debug)

## Setup Instructions

To use the templates and script in this folder, first decide if you want your
audit logs saved in the same project as the hosted data (local audit logs), or
in a separate project (remote audit logs). Remote audit logs can be especially
beneficial if you have several data projects.

1.  [Complete Script Prerequisites](#script-prerequisites) the first time using
    these scripts.
1.  [Create Groups](#create-groups) for the dataset and audit logs (if using
    remote audit logs) project(s).
1.  [Create a YAML config](#create-a-yaml-config) for the project(s) you want to
    deploy.
1.  [Create New Projects](#create-new-projects) using the `create_project.py`
    script. This will create the audit logs project (if required) and all data
    hosting projects.

### Script Prerequisites

NOTE: If running through Cloud Shell, all of the following dependencies are
already available.

-   [Go 1.10+](https://golang.org/dl/)

-   [Python 3.6+](https://www.python.org/downloads/)

-   [Bazel 0.25+](https://docs.bazel.build/versions/master/install.html)

-   [Gcloud SDK](https://cloud.google.com/sdk/install)

-   [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

-   [Pip](https://pip.pypa.io/en/stable/installing/)

### Python Dependencies

Install Python dependencies with the following command:

```shell
$ pip3 install -r requirements.txt
```

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
`create_project` script. This flag will soon become default and the old style
configs will be deprecated. Old style config examples can be found
[here](https://github.com/GoogleCloudPlatform/healthcare/tree/5bfa6b72a8077028ead4ff8c498325915180c3b8/deploy/samples).

### Create New Projects

Use the `create_project.py` script to create an audit logs project (if using
remote audit logs) and one or more data hosting projects.

1.  Make sure the user running the script is in the owners group(s) of all
    projects that will be created, including the audit logs project (if used).
    You should remove the user from these groups after successful deployment.
1.  If not already logged in, run `gcloud init` to log in as a user with
    permission to create projects under the specified organization and billing
    account
1.  If you provided a `stackdriver_alert_email` in any project, then when
    prompted during the script, follow the instructions to create new
    Stackdriver Accounts for these projects.
1.  If you provided a `forseti` config and the project hasn't been deployed you
    may be prompted for additional steps during the Forseti instance
    installation.
1.  Optional: pass a `--projects` flag listing the projects you wish to deploy,
    or `*` (default) to deploy all projects.

    WARNING: deploying a project that was previously deployed will trigger an
    update.

1.  If the projects were deployed successfully, the script will write a YAML
    file at `--output_yaml_path`, containing a `generated_fields` block for each
    newly-created project. These fields are used to generate monitoring rules.
    Update the original YAML config to include this block for the new
    project(s). We recommend setting `--output_yaml_path` to match
    `--project_yaml`, unless your input YAML is a template with environment
    variables that you intend to re-use.

    WARNING: if the script failed at any step, please sync `--output_yaml_path`
    (if it exists) with the input file before trying again.

```shell
$ git clone https://github.com/GoogleCloudPlatform/healthcare
$ cd healthcare
# Note: --incompatible_use_python_toolchains is not needed after Bazel 0.27
# See https://github.com/bazelbuild/bazel/issues/7899.
$ bazel run --incompatible_use_python_toolchains deploy:create_project -- --project_yaml=${PROJECT_CONFIG?} --projects=${PROJECTS?} --output_yaml_path=${PROJECT_CONFIG?} --nodry_run
```

If the script fails at any point, try to correct the error, sync the output yaml
file with the input and try again.

### Disabled Unneeded APIs

NOTE: This will be moved to `create_project.py`.

List the APIs that are enabled for your project, and remove any that you no
longer require:

```shell
gcloud services list --project ${PROJECT_ID?}

...

gcloud services --project ${PROJECT_ID?} disable ${SERVICE_NAME}
```

## Updates

You may wish to modify a project to add additional resources or change an
existing setting. The `create_project.py` script can be used to also update a
project. Listing a previously deployed project in the `--projects` flag (or
setting the flag to be `"*"` for all projects), will trigger an update.

WARNING: Deployment Manager will run with deletion policy "ABANDON". Thus, if a
resource is removed from a project, it will become unmonitored rather than
deleted. It is the responsibility of the user to manually remove stale
references including IAM bindings and enabled APIs.

## Steps

The script `create_project.py` takes in YAML config files and creates one or
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

1.  Creates Stackdriver Alerts for IAM changes and unexpected GCS bucket access.

1.  If a `forseti` block is defined:

    *   Runs the Forseti installer to deploy a Forseti instance (user may be
        prompted during installation)
    *   Grants permissions for each project to the Forseti service account so
        they may be monitored.
    *   Generates Forseti rules and writes them to the Forseti server bucket.

## Resources

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
forseti                 | [Forseti Installer](https://github.com/forseti-security/forseti-security/tree/dev/install)
gce_firewall            | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/firewall)
gce_instance            | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/instance)
gcs_bucket              | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/gcs_bucket)
gke_cluster             | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/gke)
gke_workload            | [Kubectl](https://kubernetes.io/docs/tutorials/configuration)
iam_custom_role         | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/iam_custom_role)
iam_policie             | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/iam_member)
pubsub                  | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/pubsub)
vpc_network             | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/network)
service_account (ALPHA) | [Deployment Manager (Direct)](https://cloud.google.com/iam/reference/rest/v1/projects.serviceAccounts/create)

## Debug

These are solutions to non-typical problems encountered:

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
