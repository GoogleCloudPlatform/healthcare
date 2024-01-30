This utility provides a sample Beam pipeline to migrate data from a GCS bucket into an HDE GCS bucket. This is a common scenario as customers often use a GCS bucket as a Cloud landing zone, but need to migrate files from that landing zone into HDE (another GCS bucket).

To use this utility:

Copy paste the Beam pipeline migration code (make sure to run `pip install apache-beam[gcp]`), and modify to suit your needs for the migration.

If you are able to upload your schema.json file into the Cloud Landig Zone GCS Bucket, use the gcloud commands in the utility to migrate the file into the HDE GCS bucket. Note here that the pipeline includes a specific GCS folder structure, necessary for the HDE GCS bucket.

Write a blank file titled `START_PIPELINE` by using the start_pipeline Beam pipeline. Note here that the pipeline includes a specific GCS folder structure, necessary for the HDE GCS bucket.