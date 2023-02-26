load("//starlark:config_test.bzl", "config_test")

package(
    default_applicable_licenses = ["//:license"],
    default_visibility = ["//visibility:public"],
)

licenses(["notice"])

[config_test(
    name = "config_test_%s" % r,
    config = "%s.yaml" % r,
    enable_terraform = True,
    deps = [
        ":generated_fields.yaml",
        "@terraform_google_forseti//:all_files",
    ],
) for r in [
    "config",
    "forseti",
    "google_cloud_bigquery_dataset",
    "google_cloud_bigtable",
    "google_cloud_dataflow",
    "google_cloud_datalab",
    "google_cloud_dataproc",
    "google_cloud_pubsub",
    "google_cloud_service_account",
    "google_cloud_spanner",
    "google_cloud_sql",
    "google_cloud_storage_bucket",
    "google_compute_firewall",
    "google_compute_image",
    "google_compute_instance",
    "google_compute_router_and_nat",
    "google_healthcare_dataset_and_datastores",
    "monitoring_notification_channels",
]]
