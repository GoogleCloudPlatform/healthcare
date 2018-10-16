# BigQuery UI

The aim of this tutorial is to get you familiarized with BigQuery web UI to query/filter/aggregate/export data.

## Prerequisites

* You will need to have a valid Google account to be able to log in to Google Cloud Platform. If you do not have one, you can create one at https://accounts.google.com. If you will be also accessing restricted datasets, you may need to notify datathon organizers to register your account for data access.

## Executing Queries

The data-hosting project `bigquery-public-data` has read-only access, as a result, you need to set a default project that you have BigQuery access to. A shared project should be created by the organizers, visit **https://bigquery.cloud.google.com/welcome/** and search for the project to access the BigQuery web interface, all the queries in this tutorial will be run through this web interface.

Note that during the datathon, all participants will be divided into teams and a Google Cloud project will be created for each team specifically. That project would be the preferred project to use. For now we'll stick with the shared project for the purpose of the tutorial.

# TLDR

In this section we are going to run a query to briefly showcase BigQuery's capability. The goal is to understand the payments made for a specific type of inpatient procedure.

Run the following query from BigQuery web interface (See "Executing Queries" section above for how to access BigQuery web interface).

![Run a query](images/run_query.png)

```SQL
#standardSQL
WITH costs AS (
  SELECT
    CAST(FLOOR(average_medicare_payments / 10000) AS INT64) AS average_cost_bucket_in_ten_thousands
    FROM `bigquery-public-data.cms_medicare.inpatient_charges_2015`
    WHERE drg_definition = '001 - HEART TRANSPLANT OR IMPLANT OF HEART ASSIST SYSTEM W MCC')
SELECT
  COUNT(average_cost_bucket_in_ten_thousands) AS number_of_procedures,
  average_cost_bucket_in_ten_thousands
  FROM costs
  GROUP BY average_cost_bucket_in_ten_thousands
  ORDER BY average_cost_bucket_in_ten_thousands ASC
```

You can download the returned result as a CSV file and genereate a chart with your preferred tools.

![Download result as a CSV file](images/save_csv.png)

The following is a scatter chart plotted from the result with Google Sheets.

![Scatter chart](images/scatter_chart.png)

# Detailed Tutorial

## BigQuery Basics

Feel free to skip this section if you are already familiar with BigQuery.

### BigQuery Table Name

A BigQuery table is uniquely identified by the three-layer hierarchy of project ID, dataset ID and table name. For example in the following query:

```SQL
SELECT
  drg_definition
FROM
  `bigquery-public-data.cms_medicare.inpatient_charges_2015`
LIMIT 10
```

`bigquery-public-data.cms_medicare.inpatient_charges_2015` specifies the table we are querying, where `bigquery-public-data` is the project that hosts the datasets, `cms_medicare` is the name of the dataset, and `inpatient_charges_2015` is the table name. Backticks (`) are used as there is a non-standard character (-) in the project name. If the dataset resides in the same project, you can safely omit the project name, e.g. `my-project.my_dataset.my_table` can be written as `my_dataset.my_table` instead.

### SQL Dialect

BigQuery supports 2 SQL dialects, legacy and standard. During this datathon we highly recommend using standard SQL dialect.

**Follow the steps below to make sure the StandardSQL dialect is used**:

1. Click "COMPOSE QUERY" on top left corner;
2. Click "Show Options" below the input area;
3. Lastly, make sure "Use Legacy SQL" is **NOT** checked, and click "Hide Options".

![Uncheck "Use Legacy SQL"](images/dialect.png)

Alternatively, ["#standardSQL" tag](https://cloud.google.com/bigquery/docs/reference/standard-sql/enabling-standard-sql) can be prepended to each query to tell BigQuery the dialect you are using, which is what we used in the TLDR section above.

## Dataset Basics

### Dataset Exploration

As mentioned previously, the datasets are hosted in a different project, which can be accessed [here](https://bigquery.cloud.google.com/dataset/bigquery-public-data:cms_medicare). On the left panel, you will see the `cms_medicare` dataset, under which you will see the table names.

To view the details of a table, simply click on it (for example the `inpatient_charges_2015` table). Then, on the right side of the window, you will have to option to see the schema, metadata and preview of rows tabs.

## Analysis

Let's take a look at a few queries. To run the queries yourself, copy the SQL statement to the input area on top of the web interface and click the red "RUN QUERY" button.

Here is an example that requires joining tables to look at the costs for inpatient and outpatient services

```SQL
SELECT
  inpatient.provider_id,
  CAST(SUM(inpatient.average_total_payments) AS INT64) AS total_inpatient_payments,
  CAST(SUM(outpatient.average_total_payments) AS INT64) AS total_outpatient_payments
FROM `bigquery-public-data.cms_medicare.inpatient_charges_2015` AS inpatient
JOIN `bigquery-public-data.cms_medicare.outpatient_charges_2015` AS outpatient
ON (inpatient.provider_id = outpatient.provider_id)
GROUP BY inpatient.provider_id
```

Let's save the result of previous query to an intermediate table for later analysis:

1. Create a dataset by clicking the caret below the search box on the left sidebar, and choose "Create new dataset";
  * Set dataset ID to "temp" and data expiration to 2 days;
  * Click "OK" to save the dataset.
2. Click "Save to table" button on the right;
  * Set destination dataset to "temp" and table to "aggregated_costs", use the default value for project;
  * Click "OK" to save the table, it usually takes less than a few seconds for demo tables.

![Create a dataset](images/create_dataset.png)

If you prefer using other tools to process the final result, a CSV file can be downloaded by clicking the "Download as CSV" button. If downloading fails because the file is too large (we highly recommend aggregating the data to a small enough result before downloading though), you can save it to a temporary table, click the caret then "Export table" button from the dropdown menu and save it to Google Cloud Storage, then you can download the file from [GCS](https://console.cloud.google.com/storage).

![Save to Google Cloud Storage](images/save_to_gcs.png)

Congratulations! You've finished the BigQuery web UI tutorial. In this tutorial we demonstrate how to query, filter, aggregate data, and how to export the result to different locations through BigQuery web UI. If you would like to explore more data, simply replace `bigquery-public-data.cms_medicare` with the project and dataset name you are interested in. Please take a look at more comprehensive examples [here](bigquery_colab.ipynb) such as creating charts.
