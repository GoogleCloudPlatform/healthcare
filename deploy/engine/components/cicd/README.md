This directory defines resources needed to setup CICD pipelines of Terraform
configs.

The CI and CD pipelines use
[Google Cloud Build](https://cloud.google.com/cloud-build) and
[Cloud Build Triggers](https://cloud.google.com/cloud-build/docs/automating-builds/create-manage-triggers)
to detect changes in the repo, trigger builds and run the workloads.

## Setup

1.  In the Terraform Engine config, add a `CICD` block under the `foundation`
    recipe and specify the following attributes:

    *   `PROJECT_ID`: Project ID of the `devops` project
    *   `STATE_BUCKET`: Name of the state bucket
    *   `REPO_OWNER`: GitHub repo owner
    *   `REPO_NAME`: GitHub repo name
    *   `BRANCH_REGEX`: Regex of the branches to set the Cloud Build Triggers to
        monitor
    *   `CONTINUOUS_DEPLOYMENT_ENABLED`: Whether or not to enable continuous
        deployment of Terraform configs
    *   `TRIGGER_ENABLED`: Whether or not to enable all Cloud Build Triggers
    *   `TERRAFORM_ROOT`: Path of the directory relative to the repo root
        containing the Terraform configs

1.  Generate the CICD Terraform configs and Cloud Build configs using the
    Terraform Engine.

1.  Before deployment CICD Terraform resources, follow
    [installing_the_cloud_build_app](https://cloud.google.com/cloud-build/docs/automating-builds/create-github-app-triggers#installing_the_cloud_build_app)
    to install the Cloud Build app and connect your GitHub repository to your
    Cloud project. This currently cannot be done through automation.

1.  Once the GitHub repo is connected, run the following commands in this
    directory to enable necessary APIs, grant Cloud Build Service Account
    necessary permissions and create Cloud Build Triggers:

    ```
    $ terraform init
    $ terraform plan
    $ terraform apply
    ```

    Two presubmit triggers are created by default and results are posted in the
    Pull Request. Failing these presubmits will block Pull Request submission.

    1.  `tf-validate`: Perform Terraform format and syntax check.
    1.  `tf-plan`: Generate speculative plans to show a set of possible changes
        if the pending config changes are deployed.

    If `CONTINUOUS_DEPLOYMENT_ENABLED` is set to `true` in your Terraform Engine
    config, `continuous_deployment_enabled` will be set to `true` in
    `terraform.tfvars` in this directory to create an additional Cloud Build
    Trigger and grant the Cloud Build Service Account broder permissions to
    automaticaly apply the config changes to GCP after the Pull Request is
    approved and submitted.

    After the triggers are created, to temporarily disable or re-enable them,
    set the `trigger_enabled` in `terraform.tfvars` to `false` or `true` and
    apply the changes by running:

    ```
    $ terraform init
    $ terraform plan
    $ terraform apply
    ```

## Operation

### Continuous Integration (presubmit)

Presubmit Cloud Build results will be posted as a Cloud Build job link in the
Pull Request, and they will be configured to block Pull Request submission.

Every new push to the Pull Request at the configured branches will automatically
trigger presubmit runs. To manually re-trigger CI jobs, comment `/gcbrun` in the
Pull Ruquest.

### Continuous Deployment (postsubmit)

Postsubmit Cloud Build job will automatically start when a Pull Ruquest is
submitted to a configured branch. To view the result of the Cloud Build run, go
to https://console.cloud.google.com/cloud-build/builds and look for your commit
to view the Cloud Build job triggered by your merged commit.

The Postsubmit Cloud Build Trigger monitors and deploys changes made to `org/`
folder only. Other changes made to `bootstrap`, `cicd` and `secrets` folders
should be deployed manually if needed.
