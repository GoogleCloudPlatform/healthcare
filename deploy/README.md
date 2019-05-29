# Google Cloud Healthcare Data Protection Toolkit

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
    file at `--generated_fields_path`, containing a block for each newly-created
    project. These fields are used to generate monitoring rules. If you will
    share this environment with other teammates we recommend checking this
    generated_fields file into a version control system.

```shell
$ git clone https://github.com/GoogleCloudPlatform/healthcare
$ cd healthcare
# Note: --incompatible_use_python_toolchains is not needed after Bazel 0.27
# See https://github.com/bazelbuild/bazel/issues/7899.
$ bazel run --incompatible_use_python_toolchains deploy:create_project -- --project_yaml=${PROJECT_CONFIG?} --projects=${PROJECTS?} --generated_fields_path=${PROJECT_CONFIG?} --nodry_run
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

## Deployment Manager Templates

There are the following templates in this folder:

*   `data_project.py` will set up a new project for hosting datasets.

*   `remote_audit_logs.py` will add BigQuery datasets or GCS buckets to hold
    audit logs in a separate project.

*   `gce_vms.py` will create Google Compute Engine VM instances and firewall
    rules.

There is also a helper script, `create_project.py` which will handle the full
creation of multiple dataset and optional audit logs project using a single YAML
config file. This is the recommended way to use the templates.

### Template data_project.py

The Deployment Manager template `templates/data_project.py` will perform the
following steps on a new project:

*   Replaces the project owners user with an owners group.
*   Creates BigQuery datasets for data, with access control.
*   Creates GCS buckets with access control, versioning, logs bucket and
    (optional) logs-based metrics for unauthorized access.
*   Creates a Cloud Pubsub topic and subscription with the appropriate access
    control.
*   If using local audit logs:
    *   Creates a BigQuery dataset to hold audit logs, with appropriate access
        control.
    *   Creates a logs GCS bucket, with appropriate access control and TTL.
*   Creates a log sink for all audit logs.
*   Creates a logs-based metric for IAM policy changes.
*   Enables data access logging on all services.

See `data_project.py.schema` for details of each field. This template is used by
the `create_project.py` script.

### Template remote_audit_logs.py

The Deployment Manager template `templates/remote_audit_logs.py` will perform
the following steps on an existing audit logs project:

*   If `logs_bigquery_dataset` is specified, creates a BigQuery dataset to hold
    audit logs, with appropriate access control.
*   If `logs_gcs_bucket` is specified, creates a logs GCS bucket, with
    appropriate access control and TTL.

See `remote_audit_logs.py.schema` for details of each field. This template is
used by the `create_project.py` script.

### Template gce_vms.py

The Deployment Manager template `templates/gce_vms.py` will perform the
following steps on an existing project:

*   Creates a new GCE VM instance for each instance listed in `gce_instances`.
*   If an instance has `start_vm` set to `True`, it is left running, otherwise
    it is stopped.
*   Creates a firewall rule for each rule listed in `firewall_rules` (if any).
    The format of these rules is the same as the
    [firewall resource](https://cloud.google.com/compute/docs/reference/rest/v1/firewalls)
    in the Compute Engine API.

See `gce_vms.py.schema` for details of each field. This template is used by the
`create_project.py` script if a project config includes `gce_instances`.

### Script create_project.py

The script `create_project.py` takes in a single YAML file and creates one or
more projects. It creates an audit logs project if `audit_logs_project` is
provided, and then creates a data hosting project for each project listed under
`projects`. For each new project, the script performs the following steps:

*   Creates a new GCP project.
*   Enables billing on the project.
*   Enables Deployment Manager and run the `data_project.py` template to deploy
    resources in the project, granting the Deployment Manager service account
    temporary Owners permissions while running the template.
*   (If using remote audit logs) creates audit logs in the audit logs project
    using the `remote_audit_logs.py` template.
*   If `gce_instances` is provided:
    *   If any VM includes a `custom_boot_image`, creates a new Compute Engine
        image using the GCS path specified.
    *   Creates new GCE VMs with SSH access enabled and, if provided in
        `firewall_rules`, creates firewall rules.
*   Prompts the user to create a Stackdriver account (currently this must be
    done using the Stackdriver UI).
*   Creates Stackdriver Alerts for IAM changes and unexpected GCS bucket access.
*   If a `forseti` block is defined:
    *   Runs the Forseti installer to deploy a Forseti instance (user may be
        prompted during installation)
    *   Grants permissions for each project to the Forseti service account so
        they may be monitored.
    *   Generates Forseti rules and writes them to the Forseti server bucket.

# Resources

Resources can be added in the config in a uniform way, but may use different
tools underneath to do the actual deployment. Since each resource may have a
very large and complex schema, we cannot cover all of it in our tooling layers.
Thus, we only implement a subset and allow the users to set additional allowed
fields as they wish. This is done through the `properties` block available for
most resources. See documentation below for each resource.

See the `samples/` directory for sample resource definitions.

NOTE: project_config.yaml.schema provides documentation on our subset. Please
reference it before adding your resource to your config.

Resource        | Deployment Tool
--------------- | ---------------
bq_dataset      | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/bigquery)
enabled_api     | [Gcloud](https://cloud.google.com/sdk/gcloud/reference/services/enable)
forseti         | [Forseti Installer](https://github.com/forseti-security/forseti-security/tree/dev/install)
gce_firewall    | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/firewall)
gce_instance    | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/instance)
gcs_bucket      | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/gcs_bucket)
gke_cluster     | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/gke)
gke_workload    | [Kubectl](https://kubernetes.io/docs/tutorials/configuration)
iam_custom_role | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/iam_custom_role)
iam_policie     | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/iam_member)
pubsub          | [Deployment Manager (CFT)](https://github.com/GoogleCloudPlatform/cloud-foundation-toolkit/tree/master/dm/templates/pubsub)
service_account | [Deployment Manager (Direct)](https://cloud.google.com/iam/reference/rest/v1/projects.serviceAccounts/create)
