``` Use one of the following to migrate your schema.json file into the HDE GCS folder.```

```gcloud storage cp gs://PREFIX-ENVIRONMENT-csv-input/CSV_DATA_SOURCE/CSV_DATA_SOURCE_{TIMESTAMP}/schemas/```

OR

```gcloud storage mv gs://SOURCE_BUCKET_NAME/SOURCE_OBJECT_NAME gs://PREFIX-ENVIRONMENT-csv-input/CSV_DATA_SOURCE/CSV_DATA_SOURCE_{TIMESTAMP}/schemas/```