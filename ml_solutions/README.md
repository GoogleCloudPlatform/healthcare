# ML Solutions for Healthcare

This directory contains tutorials that show how to use Google Cloud
technologies to work with structured healthcare data.

[generate_synthea_dataset.ipynb](https://github.com/GoogleCloudPlatform/healthcare/blob/master/ml_solutions/generate_synthea_dataset.ipynb)
is a Jupyter notebook that uses the [Synthea](https://github.com/synthetichealth/synthea) generator to generate synthetic data and then uses a data importer tool to upload the data into BigQuery.
This is a prerequisite for other tutorials that are based on the Synthea dataset. This tutorial should be run using [Google Colab](https://colab.research.google.com/).

[synthea_bqml_automl.ipynb](https://github.com/GoogleCloudPlatform/healthcare/blob/master/ml_solutions/synthea_bqml_automl.ipynb)
is a Jupyter notebook that uses a query builder to transform the Synthea dataset to a format ready for model training. The notebook also shows how to train linear and nonlinear models on the transformed data using BigQuery ML and AutoML Tables. This tutorial can be run using either [Google Colab](https://colab.research.google.com/) or [Google Cloud Datalab](https://cloud.google.com/datalab/).






