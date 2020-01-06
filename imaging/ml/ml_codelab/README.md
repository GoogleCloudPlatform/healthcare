# ML Codelabs on Cloud Healthcare API

This directory contains ML codelabs on top of the Cloud Healthcare API. They are intended to be run on Cloud Datalab. Follow these instructions to get started.

## Enable Cloud Datalab API

We first need to enable the Cloud Datalab API and its dependencies (Compute Engine and Cloud Source Repositories API). Follow the steps listed [here](https://cloud.google.com/datalab/docs/quickstart) to "Enable the APIS".

## Set-up permissions

We need to allow the service account running the Datalab instance to administer Pubsub changes. The codelabs utilize Cloud Pubsub as a notification mechanism. Enter your project ID below and execute the following:

```shell
PROJECT_ID=<YOUR PROJECT_ID>

gcloud config set project ${PROJECT_ID}
PROJECT_NUMBER=`gcloud projects describe ${PROJECT_ID} | grep projectNumber | sed 's/[^0-9]//g'`
COMPUTE_ENGINE_SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
gcloud projects add-iam-policy-binding ${PROJECT_ID} --member "serviceAccount:${COMPUTE_ENGINE_SERVICE_ACCOUNT}" --role roles/pubsub.admin
```
## Create Datalab instance

This following shell command will create a SSH tunnel allowing you to visit the Datalab UI in your web browser. You can run this on a shell in your **local machine** or **[Cloud Shell](https://cloud.google.com/shell/docs/)**.

```shell
ZONE="us-central1-c"

gcloud components install datalab
datalab create mydatalab --machine-type n1-standard-4 --zone ${ZONE}
```

If it asks you to use datalab-network, you should enter **y** (yes).

When done creating the instance, the shell will print some instruction on how to navigate to the Cloud Datalab UI.

## Set-up Datalab instance

TIP: If you are running on **[Cloud Shell](https://cloud.google.com/shell/docs/)**, you can use "Web Preview" icon (square button on top right of the shell) to connect to Datalab instance at the given port. If you are running on a local machine, you can simply point your browser to the Datalab address that was given above.

In the Datalab UI, click on the top left to create a new "Notebook".

Then enter the following to import the Git repo containing the ML notebooks/code. This will clone the codelab code to your Datalab environment.

```ipython
!git clone https://github.com/GoogleCloudPlatform/healthcare.git
```

Then, click on the Google Cloud Datalab logo in the top bar to go back to the Datalab homepage. In the Datalab file UI, navigate to *datalab/healthcare/imaging/ml_codelab*.

Then, click one of the one codelabs (.ipynb files) to begin.
