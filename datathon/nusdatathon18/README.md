# Datathon Tutorials

Welcome to [NUS-NUH-MIT Datathon](http://www.nus-datathon.com/)!

We have prepared tutorials to get you started on [BigQuery](https://cloud.google.com/bigquery/), the tool to filter, join, aggregate and extract data from the raw datasets for analysis. In each of the tutorial, a couple of comprehensive examples are included to show how to view the datasets, run transformations and analyze them.

* For Python users, please start from the [Python colab](http://colab.research.google.com/github/GoogleCloudPlatform/healthcare/blob/master/datathon/nusdatathon18/tutorials/bigquery_tutorial.ipynb) (a copy is available in the [tutorials](tutorials/bigquery_tutorial.ipynb) folder as well), which is a Jupyter notebook hosted in Google Drive, and can be shared with other people for collaboration. It has the most comprehensive examples, including how to train machine learning models on the MIMIC demo dataset with [Tensorflow](https://www.tensorflow.org/).
* For people who has experience with R, checkout our [R tutorial](tutorials/bigquery_tutorial.Rmd), which provides an interactive interface to go through the tutorial in RStudio.
  * Please note that a copy of this tutorial is already included in the RStudio servers running on both the shared and private cloud projects. All you need to do is finding the external IP address of the VM instance under "External IP" column from the shared or private project (see details below), and access RStudio by visiting http://EXTERNAL_IP:8787 (e.g. http://35.192.32.36:8787) from a browser.
     * Before the datathon, please use the [shared project](https://console.cloud.google.com/compute/instances?project=nus-datathon-2018-team-00). The username and password should be in the email from the organizers. If you want to change the default password, run `passwd` from the built-in terminal in RStudio after logging in.
     * During the datathon, please use the private project assigned to your team. The usernames are `analystX' (where `X' is between 1 and 5), passwords are the same as usernames.

![Lookup external IP](tutorials/images/external_ip.png)

![RStudio terminal](tutorials/images/rstudio_terminal.png)

* If you are not familiar with either Python or R, take a look at [BigQuery web UI](tutorials/bigquery_ui.md), which requires no programming experience. With BigQuery web UI, aggregated data can be easily exported as CSV files and then processed by other tools.

In addition, we have also included [a tutorial](http://colab.research.google.com/github/GoogleCloudPlatform/healthcare/blob/master/datathon/nusdatathon18/tutorials/ddsm_ml_tutorial.ipynb) on training a simple convolutional neural network (CNN) to classify [CBIS-DDSM](https://wiki.cancerimagingarchive.net/display/Public/CBIS-DDSM) images. [Our Cloud Machine Learning Engine tutorial](tutorials/cloud_ml_engine_tutorial.md) shows how to train the same model on GPU and [TPU](https://en.wikipedia.org/wiki/Tensor_processing_unit) respectively.

Have fun!