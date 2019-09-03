# Google Cloud Project Setup Guide for Datathons

This guide is designed to help structured data users (e.g. healthcare datathon
organizers and participants, structured data analysis course instructors and
students) to set up a set of Google Cloud Projects, in order to host the
structured datasets and data analysis environment in a compliant and audited
way. More specifically, it explains how to create:

1.  An auditing project to collect audit records from the data-hosting project
    and work project created below.
1.  A data hosting project, where structured data is hosted both in its raw
    format in a Google Cloud Storage bucket, and as BigQuery tables ready for
    controlled data access.
1.  One or more work projects for running analyses. These include running
    BigQuery jobs and storing intermediate data, and running Google Compute
    Engine virtual machines for general-purpose computing.

If you are the owner of a dataset (e.g. de-identified electronic health records
data), setting up projects 1 and 2 enables data hosting. If you want to provide
and share a cloud environment for a group of people to run analysis, projects 1 and
3 are needed. If you want to both host the data and provide an analysis
environment, then all three projects are needed.

The project setup is based on the Google Cloud Healthcare Deployment Manager
Templates. Please see its [documentation](../../deploy/README.md) for
more background information.

## Command-line Environment Setup

### Setup using Cloud Shell in Google Cloud Platform Console

You can perform all the setup through Cloud Shell in Google Cloud Platform. In
this case, many of the dependencies will be installed for you already. If
you will use Cloud Shell, skip to the next section (Billing and Cloning
Repository)

### Setup using Google Cloud SDK on your own machine

Follow instructions on
[Google Cloud SDK Installation Page](https://cloud.google.com/sdk/install) to
install the command-line interface. It contains `gcloud`, `gsutil`, and `bq`,
which are required for the project setup.

After you install the Google Cloud SDK, run the following command to initialize
it:

```shell
gcloud init
```

As part of the initialization, you will be asked to sign in with your Google
account and create a default project if you don't have one already. Please sign
in with the Google account that you will be using for the rest of the project
setup. As for the default project, we won't be using it. So you can ignore it or
create a dummy one.

For a first-time Google Cloud user, you need to accept the Terms of Service on
[Cloud Console](https://console.cloud.google.com) to proceed.

You can run the following commands to double check if `gsutil` and `bq` are
properly set up. For new Google Cloud SDK users, it is recommended to run the
commands, which may prompt you to set up some default configs.

```shell
gsutil ls
bq ls
```

If you do not have one yet, please visit the billing account
[page](https://cloud.google.com/billing/docs/how-to/manage-billing-account) to
create a Google Cloud billing account. It is of the form `01ABCD-234E56-F7890G1`
and needed for the project setup below.

## Billing and Cloning Repository

```shell
# The billing account ID will be like the following format.
export BILLING_ACCOUNT=01ABCD-234E56-F7890G1
```

Finally, you need to clone the open source toolkit from
[GoogleCloudPlatform/healthcare](https://github.com/GoogleCloudPlatform/healthcare)
and install Python dependencies.

```shell
git clone https://github.com/GoogleCloudPlatform/healthcare.git
cd healthcare/deploy
pip3 install -r requirements.txt
cd ..
```

## Choosing Domain Name, Project Prefix and Project Zones

If you own a GSuite domain or have opted your domain into
[Google Cloud Identity](https://cloud.google.com/identity/) (see
[instructions](domain_management.md)), you can create a cloud project within the
domain; otherwise, you can use a default google groups domain. For the projects
that we will be creating, you can choose a common prefix and the individual
projects will be named with the prefix you set.

## Permission Control Group Setup

Google Cloud uses Gmail accounts, groups in supported GSuite or Cloud Identity
domains or public [Google Groups](https://groups.google.com) for permission
control. We recommend that the project owners create a set of groups for
predefined roles, so that individual permission can be controlled easily by
group membership without modifying the cloud project. We recommend you define
the following groups, and add more as necessary. Please remember to set the
"Join the Group" config to allow only invited users and restrict the "View
Topics" permission to members of the group.

Note: the recommended naming convention for groups is
{PROJECT_PREFIX}-{ROLE}@{DOMAIN}

e.g. `foo-project-owners@googlegroups.com` for PROJECT_PREFIX=foo-project,
ROLE=owners, domain=googlegroups.com.

You will need to create the following groups:

Two organizer groups, meant for datathon organizers (for datathons) and course
administrators and lecturers (for courses) to join.

-   `OWNERS_GROUP`: Project owners group, has full permission to the projects,
    only used for initial project setup and infrequent management.
-   `AUDITORS_GROUP`: Project auditors who have the permission to view audit
    logs.
-   `DATA_EDITORS_GROUP`: Resource editors who can modify and delete data within
    data holding resources.

Two user groups, meant for datathon participants (for datathons) and students
(for courses) to join. Note that if the dataset in the data-hosting project is
only meant to be used during the event, and it is the same group of people as
the participants list that has access the data, then you can assign the same
group to both environment variables.

-   `DATA_READERS_GROUP`
-   `PROJECT_USERS_GROUP`

### G Suite or Cloud Identity
If you own a GSuite domain or a Cloud Identity-enabled domain, you may set up
the groups and membership programmatically using [this domain management guide](domain_management.md),
and then the following steps. Alternatively, you may create the groups using
[G Suite Admin Console](https://admin.google.com/AdminHome#GroupList:), and then
set the permissions as listed below in the public groups section.

```shell
# This can be used if your groups are part of a G Suite account.
# If you are using public Google Groups you will have to manually create and
# configure the groups using the Google Groups UI.
# This assumes you generated the access token in the domain management guide.
OWNERS_GROUP=<NAME_OF_YOUR_OWNERS_GROUP>
AUDITORS_GROUP=<NAME_OF_YOUR_AUDITORS_GROUP>
DATA_EDITORS_GROUP=<NAME_OF_YOUR_EDITORS_GROUP>
DATA_READERS_GROUP=<NAME_OF_YOUR_READERS_GROUP>
PROJECT_USERS_GROUP=<NAME_OF_YOUR_PROJET_USERS_GROUP>

GROUP_LIST=(${OWNERS_GROUP} ${AUDITORS_GROUP} ${DATA_EDITORS_GROUP} ${DATA_READERS_GROUP} ${PROJECT_USERS_GROUP})

for group_email in "${GROUP_LIST[@]}"
do
  curl -X POST --header "Content-Type: application/json" \
  --header "Authorization: Bearer ${TOKEN}" \
  --data "{\"email\":\"${group_email}\"}" \
  https://www.googleapis.com/admin/directory/v1/groups

  curl --request PATCH \
    "https://www.googleapis.com/groups/v1/groups/${group_email}" \
    --header "Authorization: Bearer ${TOKEN}" \
    --header 'Accept: application/json' \
    --header 'Content-Type: application/json' \
    --data '{"whoCanJoin":"INVITED_CAN_JOIN","allowExternalMembers":"true","whoCanPostMessage":"ALL_MANAGERS_CAN_POST"}' \
    --compressed
done
```

### Public Google Groups
For public Google Groups, you have to set up the groups and membership using the
[Google Groups UI](https://groups.google.com).

Create each of the aforementioned Google groups, then use the following secure
settings of basic permissions when you create them:

*   Group type: Email list
*   View Topics: All Members of Group (and Managers)
*   Post: Only Managers and Owners
*   Join the group: Only Invited Users

A note for the owners group: if you do not have a domain when creating the
following projects, then the resulting project cannot have a group as its owner.
To remedy this, we grant all members in OWNERS_GROUP the permission to change
the project's IAM setting, including add themselves as individual owners of the
project, if necessary.

### Group Owners

Add the user you are currently using as a member of the `OWNERS_GROUP` that was
created above. If you created groups within a domain, log in to the
[G Suite Admin Console](https://admin.google.com/AdminHome#GroupList:), or if
you used public Google Groups, use the [Google Groups UI](https://groups.google.com).

## Project Skeleton Setup

Once the groups are created, check out the deployment configuration file
`datathon_projects.tmpl.yaml` and its input `input.yaml`. Update the values in
`input.yaml` with your own fields.

Once the YAML config is updated and verified, please run the following command
to create the Google Cloud projects, including

*   An audit project
    *   named `${PROJECT_PREFIX}-auditing`, whose owner is set to
        `${OWNERS_GROUP}` if your domain has an organization configured;
        otherwise the owner will be the active Google account in the Google
        Cloud SDK config,
    *   with a BigQuery dataset for audit logs analysis, with read and job
        running permission granted to `${AUDITORS_GROUP}`.
*   A data-hosting project
    *   named `${PROJECT_PREFIX}-data`,
    *   whose project-level owner is set to `${OWNERS_GROUP}` if your domain has
        an organization configured; otherwise the owner will be the active
        Google account in the Google Cloud SDK config.
    *   Set the read-write editors of resources to `${DATA_EDITORS_GROUP}`.
    *   Direct audit logs to the auditing project `${PROJECT_PREFIX}-auditing`.
*   A team project
    *   named `${PROJECT_PREFIX}-team,
    *   whose permission group and auditing are set up in the same way as the
        data-hosting project.
    *   Pre-create a ${PROJECT_PREFIX}-shared-files GCS bucket for users to
        store shared files.
    *   Grant limited preset roles (project viewer, and BigQuery and Storage
        user, etc) to `${PROJECT_USERS_GROUP}`.
    *   Set up a Compute Engine VM with the free version of RStudio server, (in
        turned-off state) and opens up port 8787 for incoming connections.

Make sure you are in the `healthcare/deploy` directory.

```shell
bazel run cmd/apply:apply -- \
  --config_path=../datathon/organizer/input.yaml \
  --output_path=../datathon/organizer/output.yaml \
  --dry_run
```

Note that the `--dry_run` flag enables dry run mode, which only prints the
commands to be run for you to review. If everything looks good, you can rerun
the command without `--dry_run` instead to actually create the projects. The
file defined in the `--output_path` flag, will store the exact configuration
used for creating the projects after environment variable substitution.

```shell
bazel run cmd/apply:apply -- \
  --config_path=../datathon/organizer/input.yaml \
  --output_path=../datathon/organizer/output.yaml \
```

In case the deployment fails, please examine the error messages and make
appropriate changes and re-run the command.

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

If you do need to generate BigQuery schemas yourself, then you need to install
[Go](https://golang.org/doc/install) before running the script below.

```shell
# Set environment variables for parameter.
OWNERS_GROUP=<NAME_OF_YOUR_OWNERS_GROUP>
DATA_EDITORS_GROUP=<NAME_OF_YOUR_EDITORS_GROUP>
DATA_READERS_GROUP=<NAME_OF_YOUR_READERS_GROUP>
PROJECT_PREFIX=<ID_PREFIX_OF_YOUR_PROJECTS>
DATASET_NAME=<NAME_OF_YOUR_DATASET>
INPUT_DIR=<DIRECTORY_PATH_FOR_YOUR_GCS_GZ_FILES>
SCHEMA_DIR=[OPTIONAL_DIRECTORY_PATH_FOR_YOUR_SCHEMA_FILES]

scripts/upload_data.sh \
  --owners_group ${OWNERS_GROUP?} \
  --editors_group ${DATA_EDITORS_GROUP?} \
  --readers_group ${DATA_READERS_GROUP?} \
  --project_id ${PROJECT_PREFIX?}-datasets \
  --dataset_name ${DATASET_NAME?} \
  --input_dir ${INPUT_DIR?} \
  --schema_dir ${SCHEMA_DIR?}
```

## Summary

Now you are all good to go. You have a data-hosting project
`${PROJECT_PREFIX}-data` with data imported to both GCS and BigQuery, and
can run custom analysis from project `${PROJECT_PREFIX}-team`. All access is
properly audited, and data available for analysis and reporting in the audit
project `${PROJECT_PREFIX}-auditing`. If you are a member of
`${AUDITORS_GROUP}`, the following is an example query that you may adapt and
use (by replacing `${PROJECT_PREFIX}` and `${YYYYMMDD}`) to see who has accessed
the data in the projects that we have set up so far.

```sql
#standardSQL
SELECT
  TIMESTAMP_TRUNC(timestamp, DAY) AS date,
  protopayload_auditlog.authenticationInfo.principalEmail AS account,
  resource.labels.project_id AS project,
  resource.type AS resource,
  COUNT(*) AS num_access
FROM `${PROJECT_PREFIX}-auditing.data_audit_logs.cloudaudit_googleapis_com_data_access_${YYYYMMDD}`
GROUP BY 1, 2, 3, 4
ORDER BY 1, 2, 3, 4
```

The final step is to add the right set of people to the various permission
groups, and share with them the news that a fully setup suite of projects are
available for their use immediately!
