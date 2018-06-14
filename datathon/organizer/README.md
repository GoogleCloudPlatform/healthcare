# Google Cloud Project Setup Guide for Datathons

This guide is designed to help datathon organizers and other data owners alike
to set up a set of Google Cloud Projects, to host datasets and data analysis
environment in a compliant and audited way. By the end of this guide, you will
have created

1.   An auditing project to collect audit records from all projects.
1.   A data hosting project, where structured data is hosted both in its raw
     format in a Google Cloud Storage bucket, and as BigQuery tables ready for
     controlled data access.
1.   One or more work projects for running analyses. These include running
     BigQuery jobs and storing intermediate data, and running Google Compute
     Engine virtual machines for general-purpose computing.

## Command-line Environment Setup

The current toolkit we provide only automates cloud project setup from Linux
platform.

Follow instructions on [Google Cloud SDK Installaiton Page](https://cloud.google.com/sdk/install)
to install the command-line interface. It contains `gcloud`, `gsutil`, and
`bq`, which are required for the project setup.

After you install the Google Cloud SDK, run the following command to initialize
it:

```shelll
gcloud init
```

As part of the initialization, you will be asked to sign in with your Google
account and create a default project if you don't have one already. Please sign
in with the Google account that you will be using for the rest of the proejct
setup. As for the default project, we won't be using it. So you can ignore it
or create a dummy one.

For a first-time Google Cloud user, you need to accept the Terms of Service on
[Cloud Console](https://console.cloud.google.com) to proceed.

You can run the following commands to double check if `gsutil` and `bq` are
properly set up. For new Google Cloud SDK users, it is recommended to run the
commands, which may prompt you to set up some default configs.

```shell
gsutil ls
bq ls
```

You also need to install the `jq` binary to facilitate parsing JSON from
command line:

```shell
sudo apt-get install jq
```

Visit the billing account [page](https://cloud.google.com/billing/docs/how-to/manage-billing-account)
to create a billing account that will be used for the project setup.

```shell
# The billing account ID will be like the following format.
BILLING_ACCOUNT=01ABCD-234E56-F7890G1
```

If you want to generate BigQuery schemas yourself, then you need to install
[Go](https://golang.org/doc/install) as well. But check the `bqschemas`
folder first, which may already contain the BigQuery schemas you need.

## Choosing Domain Name and Project Prefix

If you own a GSuite domain, you can create a cloud project within the domain;
otherwise, you can use a default google groups domain. For the projects that
we will be creating, you can choose a common prefix and the individual projects
will be named with the prefix you set. For example,

```shell
DOMAIN=googlegroups.com
PROJECT_PREFIX=my-project
```

## Permission Control Group Setup

Google Cloud uses Gmail accounts, GSuite accounts or [Google Groups](https://groups.google.com)
for permission control. We recommend that the project owners create a set of
groups for predefined roles, so that individual permission can be controlled
easily by group membership without modifying the cloud project. We recommend
you define the following groups, and add more as necessary.

```shell
# Project owners group, has full permission to the projects, only used for
# initial project setup and infrequent management.
OWNERS_GROUP=${PROJECT_PREFIX}-owners@${DOMAIN}

# Project auditors who has the permission to view audit logs.
AUDITORS_GROUP=${PROJECT_PREFIX}-auditors@${DOMAIN}

# Members who have read-write access to the data hosted in the projects.
EDITORS_GROUP=${PROJECT_PREFIX}-readwrite@${DOMAIN}

# Data users who have read-only access to the data hosted in the projects.
DATA_READERS_GROUP=${PROJECT_PREFIX}-data-readers@${DOMAIN}

# Work project users who have granted access to the work project environments.
PROJECT_USERS_GROUP=${PROJECT_PREFIX}-users@${DOMAIN}
```

For each of the aforementioned google groups, please use the following secure
settings of basic permissions when you create them:

*   Group type: Email list
*   View Topics: All Members of Group (and Managers)
*   Post: Only Managers and Owners
*   Join the group: Only Invited Users

## Audit Project Setup

Create an audit project by running the following command, which will

*   Create a project named `${PROJECT_PREFIX}-auditing`, whose owner is set to
    `${OWNERS_GROUP}` if your domain has an organization configured; otherwise
    the owner will be the active Google account in the Google Cloud SDK config.
*   Create a BigQuery dataset for audit logs analysis, with read (and job
    running) permission granted to `${AUDITORS_GROUP}`.

```shell
scripts/create_audit_project.sh --owners_group ${OWNERS_GROUP} \
  --audit_project_id ${PROJECT_PREFIX}-auditing \
  --billing_account ${BILLING_ACCOUNT} \
  --auditors_group ${AUDITORS_GROUP}
```

The command will print out the flags that you need to pass in for the data
hosting project and the team projects. Set them in environment variables:

```shell
AUDIT_DATASET_ID=<DATASET_ID_OUTPUT_FROM_THE_ABOVE_SCRIPT>
```

## Data-Hosting Project Setup

Create a data-hosting project by running the following command, which will

*   Create a project named `${PROJECT_PREFIX}-datasets`
*   Set the owner of the project to `${OWNERS_GROUP}` if your domain has an
    organization configured; otherwise the owner will be the active Google
    account in the Google Cloud SDK config.
*   Set the read-write editors of the project to `${EDITORS_GROUP}`.
*   Direct audit logs to the auditing project `${PROJECT_PREFIX}-auditing`.

```shell
scripts/create_data_hosting_project.sh --owners_group ${OWNERS_GROUP} \
  --editors_group ${EDITORS_GROUP} \
  --data_hosting_project_id ${PROJECT_PREFIX}-datasets \
  --billing_account ${BILLING_ACCOUNT} \
  --audit_project_id ${PROJECT_PREFIX}-auditing \
  --audit_dataset_id ${AUDIT_DATASET_ID}
```

### Data Importing

With the data-hosting project set up in the previous step, any member of the
`${EDITORS_GROUP}` (as well as the `${OWNERS_GROUP}`, but we encourage the use
of the least privileged group) is able to import a new dataset to the
data-hosting project.

Assuming that you have a list of `.csv.gz` files containing the structured data,
where the first row consists of a comma-separated column names, and the rest of
the file are comma-separated data of the same number of columns, you can run the
following script to import all the files into a GCS bucket, as well as a
BigQuery dataset.

The BigQuery schemas will be automatically detected on the fly by the script.
Nevertheless, the schema detection is time consuming, if you have the BigQuery
schemas, you can pass the folder path by the `--schema_dir` flag. You should
always check if the `bqschemas` folder contains the desired schemas already.

```shell
# Set environment variables for parameter.
DATASET_NAME=<NAME_OF_YOUR_DATASET>
INPUT_DIR=<DIRECTORY_PATH_FOR_YOUR_GCS_GZ_FILES>
SCHEMA_DIR=[OPTIONAL_DIRECTORY_PATH_FOR_YOUR_SCHEMA_FILES]

scripts/upload_data.sh --owners_group ${OWNERS_GROUP} \
  --editors_group ${EDITORS_GROUP} \
  --readers_group ${DATA_READERS_GROUP} \
  --project_id ${PROJECT_PREFIX}-datasets \
  --dataset_name ${DATASET_NAME} \
  --input_dir ${INPUT_DIR} \
  --schema_dir ${SCHEMA_DIR}
```

## Team Environment Projects

You can create one or more team environment project for granted users to run
jobs in. This range from running a BigQuery query against the data-hosting
project, to materialize intermediate in its read-write BigQuery datasets and
GCS buckets, to running arbitrary software in a VM. As an illustration, you may
run the following script to create such a project

```shell
TEAM_PROJECT_ID=${PROJECT_PREFIX}-team-00
scripts/create_team_project.sh --owners_group ${OWNERS_GROUP} \
  --users_group ${PROJECT_USERS_GROUP} \
  --team_project_id ${TEAM_PROJECT_ID} \
  --billing_account <BILLING_ACCOUNT> \
  --audit_project_id <AUDIT_PROJECT_ID_FROM_AUDIT_PROJECT_SETUP> \
  --audit_dataset_id <AUDIT_DATASET_ID_FROM_AUDIT_PROJECT_SETUP>
```

This will

*   Create a project named `${PROJECT_PREFIX}-team-00`
*   Set the owner of the project to `${OWNERS_GROUP}` if your domain has an
    organization configured; otherwise the owner will be the active Google
    account in the Google Cloud SDK config.
*   Set the users with aforementioned access to `${PROJECT_USERS_GROUP}`.
*   Direct audit logs to the auditing project `${PROJECT_PREFIX}-auditing`.
*   Create a team file sharing Google Cloud Storage bucket.
*   Create and start a virtual machine.

## Summary

Now you are all good to go. You have a data-hosting project
`${PROJECT_PREFIX}-datasets` with data imported to both GCS and BigQuery, and
can run custom analysis from project `${PROJECT_PREFIX}-team-00`. All access is
properly audited, and data available for analysis and reporting in the audit
project `${PROJECT_PREFIX}-auditing`.

The final step is to add the right set of people to the various permission
groups, and share with them the news that a fully setup suite of projects are
available for their use immediately!
