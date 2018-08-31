# Google Cloud Healthcare Deployment Automation

A collection of templates for configuration of GCP resources to hold datasets.

*   [Setup Instructions](#setup-instructions)
    *   [Script Prerequisites](#script-prerequisites)
    *   [Create Groups](#create-groups)
    *   [Create a YAML config](#create-a-yaml-config)
    *   [Create New Projects](#create-new-projects)
    *   [Disabled Unneeded APIs](#disabled-unneeded-apis)
*   [Deployment Manager Templates](#deployment-manager-templates)
    *   [Template data_project.py](#template-data_projectpy)
    *   [Template remote_audit_logs.py](#template-remote_audit_logspy)
    *   [Script create_project.py](#script-create_projectpy)

## Setup Instructions

To use the templates and script in this folder, first decide if you want your
audit logs saved in the same project as the hosted data (local audit logs),
or in a separate project (remote audit logs). Remote audit logs can be
especially beneficial if you have several data projects.

1.  [Complete Script Prerequisites](#script-prerequisites) the first time using
    these scripts.
1.  [Create Groups](#create-groups) for the dataset and audit logs (if using
    remote audit logs) project(s).
1.  [Create a YAML config](#create-a-yaml-config) for the project(s) you want
    to deploy.
1.  [Create New Projects](#create-new-projects) using the `create_project.py`
    script. This will create the audit logs project (if required) and all data
    hosting projects.

### Script Prerequisites

To use these scripts, you will need to install `gcloud` and the Python YAML
package. If you are running these scripts from the Cloud Shell, then `gcloud` is
already installed. To install the Python YAML package in Cloud Shell, run

```shell
sudo apt install python-yaml
```

If you are using Python 3, install `python3-yaml` instead.


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
`samples/project_with_remote_audit_logs.yaml`, depending on whether you are
using remote or local audit logs.

*   The `overall` section contains organization and billing details applied to
    all projects. Omit the `organziation_id` if the projects are not being
    deployed under a GCP organization.
*   If using remote audit logs, include the `audit_logs_project` section, which
    describes the project that will host audit logs.
*   Under `projects`, list each data hosting project to create.

### Create New Projects

Use the `create_project.py` script to create an audit logs project (if using
remote audit logs) and one or more data hosting projects.

1.  If not already logged in, run `gcloud init` to log in as a user with
    permission to create projects under the specified organization and billing
    account
1.  Run `./create_project.py --project_yaml=${PROJECT_CONFIG?} --nodry_run`,
    where `PROJECT_CONFIG` is the YAML file containing your project
    description(s).
1.  If you provided a `stackdriver_alert_email` in any project, then when
    prompted during the script, follow the instructions to create new
    Stackdriver Accounts for these projects.

If the script fails at any point, try to correct the error and use the flags
`--resume_from_project=` and `--resume_from_step=` to continue from the project
and step that failed.

### Disabled Unneeded APIs

NOTE: This will be moved to `create_project.py`.

List the APIs that are enabled for your project, and remove any that you no
longer require:

```shell
gcloud services list --project ${PROJECT_ID?}

...

gcloud services --project ${PROJECT_ID?} disable ${SERVICE_NAME}
```

## Deployment Manager Templates

There are two templates in this folder:

*   `data_project.py` will set up a new project for hosting datasets.

*   `remote_audit_logs.py` will add BigQuery datasets or GCS buckets to hold
    audit logs in a separate project.

There is also a helper script, `create_project.py` which will handle the
full creation of multiple dataset and and optional audit logs project using a
single YAML config file. This is the recommended way to use the templates.

### Template data_project.py

The Deployment Manager template `templates/data_project.py` will perform the
following steps on a new project:

*   Replace the project owners user with an owners group.
*   Create BigQuery datasets for data, with access control.
*   Create GCS buckets with access control, versioning, logs bucket and
    (optional) logs-based metrics for unauthorized access.
*   Create a Cloud Pubsub topic and subscription with the appropriate access
    control.
*   If using local audit logs:
    *   Create a BigQuery dataset to hold audit logs, with appropriate access
        control.
    *   Create a logs GCS bucket, with appropriate access control and TTL.
*   Create a log sink for all audit logs.
*   Create a logs-based metric for IAM policy changes.
*   Enable data access logging on all services.

See `data_project.py.schema` for details of each field. This template is used by
the `create_project.py` script.

### Template remote_audit_logs.py

The Deployment Manager template `templates/remote_audit_logs.py` will perform
the following steps on an existing audit logs project.

*   If `logs_bigquery_dataset` is specified, create a BigQuery dataset to hold
    audit logs, with appropriate access control.
*   If `logs_gcs_bucket` is specified, create a logs GCS bucket, with
    appropriate access control and TTL.

See `remote_audit_logs.py.schema` for details of each field. This template is
used by the `create_project.py` script.

### Script create_project.py

The script `create_project.py` takes in a single YAML file and creates one or
more projects. It creates an audit logs project if `audit_logs_project` is
provided, and then creates a data hosting project for each project listed under
`projects`. For each new project, the script performs the following steps:

*   Create a new GCP project.
*   Enabled billing on the project.
*   Enable Deployment Manager and run the `data_project.py` template to deploy
    resources in the project, granting the Deployment Manager service account
    temporary Owners permissions while running the template.
*   (If using remote audit logs) creates audit logs in the audit logs project
    using the `remote_audit_logs.py` template.
*   Prompts the user to create a Stackdriver account (currently this must be
    done using the Stackdriver UI).
*   Creates Stackdriver Alerts for IAM changes and unexpected GCS bucket access.
