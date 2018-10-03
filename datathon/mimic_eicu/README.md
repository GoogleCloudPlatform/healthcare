# Datathon Tutorials

Welcome to datathon on GCP!

We have prepared tutorials to get you started on [BigQuery](https://cloud.google.com/bigquery/), the tool to filter, join, aggregate and extract data from the raw datasets for analysis. In each of the tutorials, some comprehensive examples are included to show how to view the datasets, run transformations and analyze them.

* For Python users, please start from the [Python colab](http://colab.research.google.com/github/GoogleCloudPlatform/healthcare/blob/master/datathon/mimic_eicu/tutorials/bigquery_tutorial.ipynb) (a copy is available in the [tutorials](tutorials/bigquery_tutorial.ipynb) folder as well), which is a Jupyter notebook hosted in Google Drive, and can be shared with other people for collaboration. It has the most comprehensive examples, including how to train machine learning models on the MIMIC demo dataset with [Tensorflow](https://www.tensorflow.org/).
* For people who have experience with R, start with the [R tutorial](tutorials/bigquery_tutorial.Rmd), which provides an interactive interface to go through the tutorial in RStudio.
  * Please note that a copy of this tutorial is already included in the RStudio servers running on both the shared and private cloud projects. All you need to do is:
      * Go to the [Console](https://console.cloud.google.com/compute/instances?) and ensure at the top that you are in the shared project;
      * Find the external IP address of the VM instance under "External IP" column (see the first screenshot below);
      * Visit http://EXTERNAL_IP:8787 (e.g. http://11.22.33.44:8787) from a browser.
          * The username and password should be in the email from the organizers. If you want to change the default password, run `passwd` from the built-in terminal in RStudio after logging in (see the second screenshot below).
          * During the datathon, please use the private project assigned to your team instead of the shared project. The usernames for your private project are `analystX` (where `X` is between 1 and 5), passwords are the same as usernames.

   ![Lookup external IP](tutorials/images/external_ip.png)

   ![RStudio terminal](tutorials/images/rstudio_terminal.png)

* If you are not familiar with either Python or R, start [BigQuery web UI](tutorials/bigquery_ui.md), which requires no programming experience. With BigQuery web UI, aggregated data can be easily exported as CSV files and then processed by other tools.

Have fun!
