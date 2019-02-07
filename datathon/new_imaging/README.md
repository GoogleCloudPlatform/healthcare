# Analytics with Medical Images

## Preparing DICOM data

If you're dealing with DICOM data, either in the form of .dcm files stored in
Google Cloud Storage (GCS) or a dicomStore in the Google Cloud Healthcare (CHC)
this is the place to start.

We've provided the script [prepare\_dicom.py](prepare_dicom.py) to automate the
first steps in ingesting DICOM data. This script can:

*   import .dcm files into a CHC dicomStore
*   export DICOM metadata from a CHC dicomStore to BigQuery
*   export DICOM image-data from a CHC dicomStore to `.png` files store on GCS

Run `python prepare_dicom.py --help` for instructions on how to run this script.

If we boil away extras like error handling, this script boils down to something
pretty compact. First we connect to CHC using the python client library

```
import cloud.healthcare as chc
client = chc.Client()
```

And choose our dataset name and dicomStore name. For example `chc_dataset_name =
'my_dataset'` and `chc_dicom_store_name = 'my_dicom_store'`.

```
dataset = client.dataset(chc_dataset_name)
dicom_store = dataset.create_dicom_store(chc_dicom_store_name)
```

Importing a glob of dicom files like `gcs_dicom_pattern =
'gs://my_bucket/**.dcm'` into your dicomStore is done like this

```
dicom_store.import_from_gcs(gcs_dicom_pattern)
```

Exporting to the metadata to a BigQuery table, and the image data to `.png`
images is also straightforward.

```
dicom_store.export_to_bigquery(bq_dataset, bq_table, overwrite_table)
dicom_store.export_to_gcs(gcs_png_path, 'image/png')
```

The rest of the code in [prepare\_dicom.py](prepare_dicom.py) handles errors,
concurrency, creating datasets, and logging, but the important parts are what we
have described here.

Here's an example of how you might run [prepare\_dicom.py](prepare_dicom.py) so
that you can access your DICOM data using CHC, BigQuery, and as `.png` files in
GCS. You can skip any of these steps (importing dcm files, exporting to BigQuery
or exporting to GCS) by omitting the corresponding command line arguments.

```
python prepare_dicom.py \
  --gcs_dicom_pattern gs://my_bucket/my_input_dataset**.dcm \
  --chc_dataset my_chc_dataset \
  --chc_dicomstore my_chc_dicomstore \
  --bq_dataset my_bigquery_dataset \
  --bq_table my_bigquery_table \
  --gcs_png_path gs://my_bucket/my_png_dataset
```

## Creating TFRecords with Cloud DataFlow

After we've used [prepare\_dicom.py](prepare_dicom.py) to create a collection of
`.png` files in GCS, the next thing we might want to do is collect these images
to a dataset that can easily be ingested by Tensorflow and Cloud ML Engine.

To this end, we've provided [png\_to\_tfrecord.py](png_to_tfrecord.py) to
convert this collection of `.png` images into a
[TFRecord](https://www.tensorflow.org/guide/datasets#consuming_tfrecord_data).

This pipeline grabs all .png files with the given path prefix on GCS, resizes
them, and writes them as tensorflow examples of the following form

```
def _bytes_feature(value):
  return tf.train.Feature(bytes_list=tf.train.BytesList(value=[value]))

example = tf.train.Example(
    features=tf.train.Features(feature={'study': _bytes_feature(study),
                                        'series': _bytes_feature(series),
                                        'instance': _bytes_feature(instance),
                                        'png_bytes': _bytes_feature(png_bytes)}))
```

Using Dataflow, this process will automatically be distributed over as many
computers as needed to obtain a timely result. Datasets with tens of millions of
images are as easy to handle as datasets with only a few images.

Here's an example of how someone might launch this pipeline on Cloud Dataflow.
After you've started this script, you can view its progress using the
[Dataflow web console](https://console.cloud.google.com/dataflow)

```
python format_input.py \
  --gcs_input_prefix  gs://my_bucket/my_png_dataset \
  --output gs://my_bucket/my_tfrecord_dataset \
  --output_image_shape 299,299,1 `#store as 299x299 black and white images` \
  --temp_location gs://my_bucket/temp \
  --project my_gcp_project \
  --requirements_file dataflow_requirements.txt \
  --runner DataflowRunner
```

## Getting batch predictions with Cloud ML engine

[build\_cloudml\_model.ipynb](build_cloudml_model.ipynb) is a jupyter notebook
that walks through building a tensorflow model and hosting it on Cloud ML
engine. It processes data in the form outputted by
[png\_to\_tfrecord.py](png_to_tfrecord.py) into feature vectors that can be
thought of as a abstract summary of the image. Go ahead and investigate it!

## Loading Batch Predictions Into BigQuery

Cloud ML Engine's batch prediction mode outputs a text file with one prediction
per line formatted as JSON. For your convenience, we have provided
[load\_feature\_vectors.py](load_feature_vectors.py), which defines a Dataflow
pipeline for loading these feature vectors into a BigQuery table.

The code is pretty simple

```
with beam.Pipeline(options=pipeline_options) as p:
_ = (
      p
      | beam.io.ReadFromText(text_input)
      | beam.ParDo(Parse(inferred_schema))
      | beam.io.gcp.bigquery.WriteToBigQuery(
          table=bq_table,
          dataset=bq_dataset,
          schema=bq_schema,
          write_disposition=BigQueryDisposition.WRITE_TRUNCATE))
```

Which means:

*   read every line of the file in parallel into a collection of strings
*   parse each string into a dictionary that maps column names to values (the
    `Parse` class is defined elsewhere in this file).
*   write the results to a new BigQuery table.

Here's an example outlining how to use it. As before, you can watch the progress
of the pipeline on the
[Dataflow web console](https://console.cloud.google.com/dataflow).

```
python load_feature_vectors.py \
  --input_text_file feature_vector_prediction.txt `# from build_cloudml_model.ipynb` \
  --bq_dataset my_bigquery_dataset \
  --bq_table image_feature_vectors \
  --temp_location gs://my_bucket/temp \
  --project my_gcp_project \
  --runner DataflowRunner
```
