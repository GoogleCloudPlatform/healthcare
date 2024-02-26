# DataFlow Pub/Sub to Healthcare API FHIR Store Streaming Pipeline

The goal of this DataFlow streaming pipeline (Classic standalone deployment) is to consume messages via Google Pub/Sub, transform raw JSON messages to FHIR resource as per [U.S Core Implementation Guide](https://build.fhir.org/ig/HL7/US-Core/) (Preferential) and POST new created FHIR resources to Healthcare API (HCAPI R4 FHIR Store) for further reconciliation ingestion via Heathcare Data Engine or standalone FHIR Store which can be used for downstream application.  
This solution is built using Google Cloud tools and services, such as Google Cloud Dataflow, Google Cloud Pub/Sub, Google Cloud Healthcare Api and Google Cloud Storage. This pipeline will help users accelerate deploying streaming data pipelines from Google Pub/Sub to Google Healthcare API FHIR Store enabling users to transform their raw data to FHIR resources where streaming application is applicable.  


# Architecture for the Pipeline is shown below
 
 ![Log output image](./images/dataflow_architecture_pubsub_to_hcapi.png)

## Products/tools used for the pipeline

# Google Cloud Pub/Sub
Overview: Pub/Sub is an asynchronous and scalable messaging service that decouples services producing messages from services processing those messages.  
Usage: Vendors/Application (web UI, Mobile Apps, etc.) will send healthcare data events in JSON format to Pub/Sub topic  

# Google Cloud Dataflow
Overview: Dataflow is a managed service for executing a wide variety of data processing patterns. The documentation on this site shows you how to deploy your batch and streaming data processing pipelines using Dataflow, including directions for using service features.  
Usage: Dataflow will read raw healthcare data events in JSON format, parse and transform them into appropriate FHIR resources as per US Core Implementation Guide. For the sake of this example we will be creating a Patient FHIR resource.

# Google Cloud Healthcare API
Overview: The Cloud Healthcare API is a fully-managed service that makes it easy to access, process, and analyze healthcare data. It provides a RESTful API that supports a wide range of healthcare data formats, including FHIR, HL7v2, and DICOM.\n
The Cloud Healthcare API is a secure and reliable service that is compliant with HIPAA and HITECH. \n
Usage: HCAPI will be used to store transformed FHIR resources (in our case Patient FHIR resource)which will be made available for downstream applications or other processes.

# Google Cloud Storage
Usage: Google Cloud Storage will be used to archive raw healthcare data sent by the source as well as log error message occurred during transformation or while posting FHIR resource to HCAPI


## Prerequisites before cloning the repository
1. Create a Google Cloud Project and set up appropriate billing and IAM permissions. Refer following [link](https://developers.google.com/workspace/guides/create-project) for more information on how to create a GCP project.
2. Create a GCS Archive and Error bucket. Refer following [link](https://cloud.google.com/storage/docs/creating-buckets) for more information on how to create a GCS bucket.
3. Create a Pub/Sub topic and subscription. Refer following [link](https://cloud.google.com/pubsub/docs/create-topic#pubsub_create_topic-Console) for more information on how to create a Pub/Sub topic and subscription.
4. Create a FHIR Store using Healthcare API. Refer following [link](https://cloud.google.com/healthcare-api/docs/how-tos/fhir#healthcare-create-fhir-store-console) for more information on how to create a FHIR Store and set up necessary permissions.
5. Understanding of FHIR Resources and types of FHIR resources.Refer following [link](https://build.fhir.org/ig/HL7/US-Core/) for more information on FHIR resources and different types of FHIR resources.


# Step by Step workflow

1. Create necessary GCS Bucket, Pub/Sub topic and FHIR Store as mentioned in the Prerequisites section.  

2. We will use the following parameter values as an example,
    1. GCP Project: demo-project  
    2. GCS Bucket :-  
        1. gcs_archive_bucket - ehr-demo-archive-bucket  
        2. gcs_error_bucket - ehr-demo-error-bucket  
    3. Pub/Sub Topic: projects/demo-project/topics/ehr-portal-events  
    4. Pub/Sub Subscription (will be created by default): projects/demo-project/subscriptions/ehr-portal-events-subscription  
    5. Healthcare API Dataset: ehr-demo  
    6. Healthcare API FHIR Store: ehr-demo-fhirstore  

3. We will use the following JSON payload for our example as an input which contains [patient](https://build.fhir.org/ig/HL7/US-Core/StructureDefinition-us-core-patient.html) details such as Address, name, gender, etc which will be used for building Patient FHIR resource as per US Core Implementation Guide.  
    ```
    {"ADDRESS":"978 Turcotte Overpass","BIRTHDATE":"2004-07-04","COUNTRY":"US","CITY":"Gill","COUNTY":"Franklin County","DRIVERS":"","ETHNICITY":"nonhispanic","FIRST":"Gidget756","GENDER":"Female","HEALTHCARE_COVERAGE":3311.4399999999996,"HEALTHCARE_EXPENSES":404742.42,"Id":"7992bf94-feee-4728-9187-2c911df2819b","LAST":"Rempel203","LAT":42.63299146919133,"LON":-72.5251235198123,"MAIDEN":"","MARITAL":"","EMAIL":"sampleemail@.yahoo.com","PREFIX":"","RACE":"white","SSN":"999-67-4045","STATE":"Massachusetts","SUFFIX":""}  
    ```  

4. The raw patient data in JSON format will transformed into the final Patient FHIR resource attached below  
    ```
    {
        "address": [
            {
                "city": "Gill",
                "country": "US",
                "district": "Franklin County",
                "line": [
                    "978 Turcotte Overpass"
                ],
                "state": "Massachusetts",
                "type": "physical",
                "use": "home"
            }
        ],
        "birthDate": "2004-07-04",
        "gender": "female",
        "id": "3820ec4a-ec08-48cd-969a-1dd9b6a32d7f",
        "identifier": [
            {
                "system": "SSN",
                "value": "999-67-4045"
            }
        ],
        "meta":{
            "lastUpdated": "2024-01-08T03:31:59.086617+00:00",
            "versionId": "MTcwNDY4NDcxOTA4NjYxNzAwMA"
        },
        "name": [
            {
                "family": "Rempel203",
                "given": [
                    "Gidget756"
                ],
                "use": "official"
            }
        ],
        "resourceType": "Patient",
        "telecom": [
            {
                "system": "email",
                "use": "home",
                "value": "sampleemail@.yahoo.com"
            }
        ]
    }
    ```  
5. Before triggering Dataflow Job, Refer to the following [link](https://cloud.google.com/dataflow/docs/quickstarts/create-pipeline-python) on how to set up a GCP environment for running a Dataflow job using Python.  

6. Below mentioned python command shows an example of triggering a dataflow streaming job with pre-defined parameters and values set an example  
    ```
    python3 -m dataflow_pubsub_to_fhirstore \
    --runner DataflowRunner \
    --project demo-project\
    --region us-central1\
    --temp_location gs://ehr-demo-dataflow-staging-bucket/tmp/ \
    --no_use_public_ips \
    --subnet regions/us-central1/subnetworks/default\
    --read_pubsub_subscription 'projects/demo-project/subscriptions/ehr-portal-events-subscription'\
    --hcapi_project_id 'demo-project'\
    --hcapi_dataset 'ehr-demo'\
    --hcapi_version 'v1'\
    --hcapi_location 'us-central1'\
    --hcapi_get_fhir_store 'ehr-demo-fhirstore'\
    --hcapi_post_fhir_store 'ehr-demo-fhirstore'\
    --gcs_archive_bucket 'ehr-demo-archive-bucket'\
    --gcs_error_bucket 'ehr-demo-error-bucket'\
    --max_num_workers 5\
    --streaming True\
    --save_main_session True\
    --requirements_file requirements.txt
    ```  
7. After triggering the job the dataflow will generate a Dataflow DAG

    ![Log output image](./images/dataflow_dag_pubsub_to_hcapi.png)

8. Publish the JSON message containing patient details mentioned in step1.  
    Below mentioned examples show how to publish messages along with sample attributes (topic and timestamp).  
    **NOTE: attribute topic and timestamp are mandatory else the code will throw errors and data won’t be transformed**

    1. Navigate to your  new topic created in pub/sub  
    2. Click on messages and click on Publish message as shown in the screenshot  
          
        ![Log output image](./images/sc1.png)

    3. Copy paste the payload containing patient details as a message body along with attributes topic and timestamp as shown in the screenshot and hit publish message.  
          
        ![Log output image](./images/sc2.png)  

9. Dataflow will read the message from Pub/Sub, transform the message, archive it into GCS bucket, build a Patient FHIR resource as per defined transformation and post the message to HCAPI as a Patient FHIR resource.
10. Below attach screenshot shows the patient FHIR resource posted to HCAPI
      
    ![Log output image](./images/sc3.png)  

11. Drain the pipeline and stop the Dataflow job to avoid unwanted utilization of resources.

  
**Note: The above example shows the leveraging Dataflow job to transform raw events, build FHIR resources and POST them to HCAPI. Please update/edit the code based on your custom transformations and business requirements.**  

