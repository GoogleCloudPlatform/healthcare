# HL7v2 to FHIR Reference Implementation

This program demonstrates how to set up a GKE pod that helps automatically tranform HL7v2 messages to FHIR resources when you work with Google Cloud Healthcare API. The deployed pod listens for pubsub notifications from HL7 stores, converts HL7v2 messages to FHIR resources and uploads the
resources to FHIR stores.

Currently only patient admission (ADT_A01) and observation results (ORU_R01) messages are supported.

## Requirements

* A [Google Cloud project](https://cloud.google.com).
* A [Docker](https://docs.docker.com/) repository. The following instructions assume the use of [Google Container Registry](https://cloud.google.com/container-registry/).
* Installed [gcloud](https://cloud.google.com/sdk/gcloud/) and [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command line tools.

## Build

We use [gradle](https://gradle.org/) as the build tool.

Run the following commands to build the docker image:

```bash
gradle dockerBuildImage
```

To run the image locally:

```bash
docker run --network=host -v ~/.config:/root/.config hl7_to_fhir_converter /healthcare/bin/healthcare --fhirProjectId=<PROJECT_ID> --fhirLocationId=<LOCATION_ID> --fhirDatasetId=<DATASET_ID> --fhirStoreId=<STORE_ID> --pubsubProjectId=<PUBSUB_PROJECT_ID> --pubsubSubscription=<PUBSUB_SUBSCRIPTION_ID> --apiAddrPrefix=<API_ADDR_PREFIX>
```

Where LOCATION_ID looks like 'us-central1', representing the zone cloud healthcare API runs in, DATASET_ID and STORE_ID are the ids of your dataset and FHIR store, respectively. You can find these in the name of your FHIR store, e.g. your FHIR store name should look like projects/<PROJECT_ID>/locations/<LOCATION_ID>/datasets/<DATASET_ID>/fhirStores/<STORE_ID>.

In the command above:
* `--network=host` is used to expose the port of the container;
* `-v ~/.config:/root/.config` is used to give the container access to gcloud credentials;

Also note that:
* `PUBSUB_PROJECT_ID` and `PUBSUB_SUBSCRIPTION_ID` are available by creating a pubsub topic and a subscription on Google Cloud;
* `API_ADDR_PREFIX` is of form `https://www.google.com:443/v1`, scheme, port and version should all be present.

## Deployment

Before deploying the docker image to GKE you need to publish the image to a registry.

```bash
docker tag hl7_to_fhir_converter gcr.io/<PROJECT_ID>/hl7-to-fhir-converter:latest
docker push gcr.io/<PROJECT_ID>/hl7-to-fhir-converter:latest
```

Next create a resource config file `converter.yaml` locally. You can use the following as a template but replace the placeholders for your use case.

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: hl7-to-fhir-converter-deployment
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: hl7-to-fhir-converter
    spec:
      containers:
        - name: hl7-to-fhir-converter
          image: gcr.io/<PROJECT_ID>/hl7-to-fhir-converter:latest
          command:
            - "/healthcare/bin/healthcare"
            - "--apiAddrPrefix=<API_ADDR_PREFIX>"
            - "--fhirProjectId=<PROJECT_ID>"
            - "--fhirLocationId=<LOCATION_ID>"
            - "--fhirDatasetId=<DATASET_ID>"
            - "--fhirStoreId=<STORE_ID>"
            - "--pubsubProjectId=<PUBSUB_PROJECT_ID>"
            - "--pubsubSubscription=<PUBSUB_SUBSCRIPTION_ID>"
```

```bash
gcloud container clusters create hl7-to-fhir-converter --scopes https://www.googleapis.com/auth/pubsub,https://www.googleapis.com/auth/cloud-healthcare
kubectl create -f converter.yaml
```

## Debug

To view the running status and logs of the pod:

```bash
kubectl get pods
kubectl logs <POD_ID>
```
