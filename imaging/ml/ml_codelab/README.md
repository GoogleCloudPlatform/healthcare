# Imaging ML Codelabs on Cloud Healthcare API

This directory contains Imaging ML Codelabs on top of the Cloud Healthcare API. They are intended to be run in an AI Platform Notebook. Follow these instructions to get started.

## Enable AI Platform Notebooks

We first need to enable the AI Platform Notebooks. Follow the steps listed [here](https://cloud.google.com/ai-platform/notebooks/docs/before-you-begin) to "Enable the APIS".

## Set-up permissions

Go to https://cloud.google.com/healthcare/docs/resources/public-datasets/tcia#cloud-healthcare-api to request permissions to tcia dataset.

**Please wait until you are granted access.** You cannot complete the codelab until granted access.

We need to allow the service account running the AI Platform Notebooks instance to administer Pubsub changes. The codelabs utilize Cloud Pubsub as a notification mechanism. Enter your project ID below and execute the following:

```shell
PROJECT_ID=<YOUR PROJECT_ID>

gcloud config set project ${PROJECT_ID}
PROJECT_NUMBER=`gcloud projects describe ${PROJECT_ID} | grep projectNumber | sed 's/[^0-9]//g'`
COMPUTE_ENGINE_SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
gcloud projects add-iam-policy-binding ${PROJECT_ID} --member "serviceAccount:${COMPUTE_ENGINE_SERVICE_ACCOUNT}" --role roles/pubsub.admin
```
## Create AI Platform Notebooks instance

Follow the steps listed [here](https://cloud.google.com/ai-platform/notebooks/docs/create-new). Create Python instance with deafult configuration 

## Create a new Notebook instance


Click "OPEN JUPYTERLAB" to open JupyterLab UI.

### Set-up environment
In the JupyterLab UI, open File -> New Launcher, and select a Terminal.
since service accounts not yet supported run following command to init gcloud with your email
```bash
gcloud init --console-only
```
select `[2] Log in with a new account` and follow instructions

### Run examples 

In the JupyterLab UI, open File -> New Launcher, and select a Python 3 Notebook.

Then run the following to import the Git repo containing the ML notebooks/code. This will clone the codelab code to your JupyterLab environment.

```ipython
!git clone https://github.com/GoogleCloudPlatform/healthcare.git
```

Then, navigate to */healthcare/imaging/ml/ml_codelab* in a left navigation bar of the JupyterLab UI.

Then click one of the two(.ipynb files) to begin.
