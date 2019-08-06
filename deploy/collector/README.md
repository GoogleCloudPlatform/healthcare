# Audit Data Collector

This directory contains an App Engine service that loads audit data
of projects created using the deployment templates, and monitored
using Forseti via Audit API.

## Setup Instructions

This App Engine service should be deployed in the project that you will be
using with the Audit Dashboard frontend.

1.  Edit `app.yaml` to set the GCS bucket that contains the project config YAML,
    Forseti rules and Forseti violations. Update other `env_variables` as
    needed.
1.  Normally, this will be the only App Engine service in the project, so it
    will be configured as the `default` service. However, if there is a
    different service in this project that you'd like to be the `default`, then:
    1.  Edit `app.yaml` to uncomment the `service: audit-data-collector` line.
    1.  Edit `cron.yaml` to uncomment the `target: audit-data-collector` line.
1.  Deploy the app:

    ```shell
    blaze build deploy/collector:app
    gcloud app deploy --skip-staging blaze-bin/deploy/collector/app
    ```

1.  The service uses the default AppEngine service account:
    `${PROJECT_ID?}@appspot.gserviceaccount.com`. This service account will need
    appropriate access configured:
    1.  Grant `roles/storage.read permission on bucket(s) defined in `app.yaml`.
    1.  (Once Audit API available) Grant `roles/healthcare.auditAdmin` for this project.
    1.  (Optional) grant `READER` permissions on all audit log datasets.
1.  Setup App Engine firewall rules to only allow Cron traffic.

    ```shell
    gcloud --project=${PROJECT_ID?} app firewall-rules update default \
        --action DENY
    gcloud --project=${PROJECT_ID?} app firewall-rules create 1000 \
        --action ALLOW --source-range 10.0.0.1 \
        --description "Allow App Engine Cron"
    ```

1.  Configure the cron job:

    ```shell
    gcloud app deploy deploy/collector/cron.yaml --project=${PROJECT_ID?}
    ```

1.  DEV ONLY: Add `ALLOW` firewall rules for trusted IP ranges to allow manually
    triggering the cron jobs (also set `CRON_REQUESTS_ONLY: false` in
    `app.yaml`), or for viewing debug pages.
