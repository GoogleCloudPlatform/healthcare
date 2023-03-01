# Imaging ML Codelabs on Cloud Healthcare API

This directory contains Imaging ML Codelabs on top of the Cloud Healthcare API. They are intended to be run in an AI Platform Notebook. Follow these instructions to get started.

## Enable AI Platform Notebooks

We first need to enable the AI Platform Notebooks. Follow the steps listed [here](https://cloud.google.com/ai-platform/notebooks/docs/before-you-begin) to "Enable the APIS".

## Set-up permissions

Go to https://cloud.google.com/healthcare/docs/resources/public-datasets/tcia#cloud-healthcare-api to request permissions to tcia dataset.

**Please wait until you are granted access.** You cannot complete the codelab until granted access.

Note: Google Cloud is managing a new version of the [TCIA dataset](https://cloud.google.com/healthcare-api/docs/resources/public-datasets/idc). Changes are needed in the notebooks/scripts to use the new dataset.

We need to allow the service account running the Datalab instance to administer roles.
To grant permission please run following commands with owner or roles/iam.securityAdmin role

```shell
PROJECT_ID=<YOUR PROJECT_ID>

gcloud config set project ${PROJECT_ID}
PROJECT_NUMBER=`gcloud projects describe ${PROJECT_ID} | grep projectNumber | sed 's/[^0-9]//g'`
COMPUTE_ENGINE_SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
gcloud projects add-iam-policy-binding ${PROJECT_ID} --member "serviceAccount:${COMPUTE_ENGINE_SERVICE_ACCOUNT}" --role roles/iam.securityAdmin
```

This can also be accomplished using the [Google Cloud Console](https://console.cloud.google.com/iam-admin/iam?project=) as well as all permission modification performed in ipynb.
## Create AI Platform Notebooks instance

Follow the steps listed [here](https://cloud.google.com/ai-platform/notebooks/docs/create-new). Create a Python instance with default configuration.

## Create a new Notebook instance


Click "OPEN JUPYTERLAB" to open JupyterLab UI.

### Run examples

In the JupyterLab UI, open File -> New Launcher, and select a Python 3 Notebook.

Then run the following to import the Git repo containing the ML notebooks/code. This will clone the codelab code to your JupyterLab environment.

```ipython
!git clone https://github.com/GoogleCloudPlatform/healthcare.git
```

Then, navigate to */healthcare/imaging/ml/ml_codelab* in a left navigation bar of the JupyterLab UI.

Then click one of the two(.ipynb files) to begin.
