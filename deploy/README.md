# Google Cloud Healthcare Deployment Automation

A collection of templates for configuration of GCP resources to hold datasets.

[TOC]

## Setup Instructions

To use the templates and script in this folder, first decide if you want your
audit logs saved in the same project as the hosted data (local audit logs),
or in a separate project (remote audit logs). Remote audit logs can be
especially beneficial if you have several data projects.

If using local audit logs:

1.  [Create Groups](#create-groups) for the dataset project.
1.  [Create a New Data Hosting Project](#create-a-new-data-hosting-project)
    using the `create_project.py` script.

If using remote audit logs:

1.  [Create Groups](#create-groups) for the audit logs project and dataset
    project(s).
1.  [Create a New Audit Logs Project](#create-a-new-audit-logs-project)
    using the `create_project.py` script.
1.  [Create a New Data Hosting Project](#create-a-new-data-hosting-project)
    using the `create_project.py` script.

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
your dataset project. We also provide recommended names based on best practices.

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

If you are using a separate Audit Logs project, then this project will also need
its own Owners Group and an Auditors group, but no data groups.

### Create a New Audit Logs project

If using remote audit logs, use the `create_project.py` script to create a new
project to host audit logs for each data hosting project:

1.  Edit a copy of the file `samples/audit_logs_project_config.yaml`, filling in
    all fields with desired values for your audit logs project.
1.  If not already logged in, run `gcloud init` to log in as a user with
    permission to create projects under the specified organization and billing
    account
1.  Run `./create_project.py --project_yaml=${PROJECT_CONFIG?} --nodry_run`,
    where `PROJECT_CONFIG` is the yaml file created in step 1.
1.  If you provided a `stackdriver_alert_email`, then when prompted during the
    script, follow the instructions to create a new Stackdriver Account.

### Create a New Data Hosting Project

Use the `create_project.py` script to create each new project that will host
datasets.

1.  Edit a copy of the file `samples/data_hosting_project_config.yaml`, filling
    in all fields with desired values for your data hosting project.
    *  If using remote audit logs, fill in the `remote_audit_logs` section.
    *  If using local audit logs, remove the `remote_audit_logs` section, and
       uncomment the `local_audit_log` section under `project_config`.
1.  If not already logged in, run `gcloud init` to log in as a user with
    permission to create projects under the specified organization and billing
    account
1.  Run `./create_project.py --project_yaml=${PROJECT_CONFIG?} --nodry_run`,
    where `PROJECT_CONFIG` is the yaml file created in step 1.
1.  If you provided a `stackdriver_alert_email`, then when prompted during the
    script, follow the instructions to create a new Stackdriver Account.

If the script fails at any point, try to correct the error and use the flag
`--resume_from_step=` to continue from the step that failed.

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
full creation of a dataset or audit log project using a single YAML config file.
This is the recommended way to use the templates.

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

*   If `logs_bigquery_dataset_ specified, create a BigQuery dataset to hold
    audit logs, with appropriate access control.
*   If `logs_gcs_bucket` specified, create a logs GCS bucket, with appropriate
    access control and TTL.

See `remote_audit_logs.py.schema` for details of each field. This template is
used by the `create_project.py` script.

### Script create_project.py

The script `create_project.py` takes in a single YAML file and performs the
following steps:

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
