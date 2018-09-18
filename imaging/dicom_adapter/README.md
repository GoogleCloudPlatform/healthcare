# DICOM Adapter

The DICOM adapter is a set of components that translate between traditional DICOM DIMSE protocols (e.g., C-STORE) and the RESTful DICOMweb protocols (e.g., STOW-RS). There are
two components, namely import and export adapter.


## Import Adapter

The Import Adapter converts incoming DIMSE (C-STORE) requests to DICOMWeb (STOW-RS) and sends them to peer.

For the list of command line flags, see [here](import/src/main/java/com/google/cloud/healthcare/imaging/dicomadapter/Flags.java)

## Export Adapter

The Export Adapter listens to [Google Cloud Pub/Sub](https://cloud.google.com/pubsub/)
for new instances, fetches them using WADO-RS, then sends them to the client.
This binary can be configured to output either C-STORE or STOW-RS via command
line flags.

To use [Google Cloud Pub/Sub](https://cloud.google.com/pubsub/), you require a [Google Cloud project](https://cloud.google.com). Furthermore, [Cloud Pubsub API](https://console.cloud.google.com/apis/api/pubsub.googleapis.com/overview) must be enabled in your Google project. The binary expects that each Cloud Pub/Sub notification consists of the WADO-RS path for the DICOM instance that is to be exported (e.g. `/studies/<STUDY_UID>/series/<SERIES_UID>/instances/<INSTANCE_UID>`).

For the list of command line flags, see [here](export/src/main/java/com/google/cloud/healthcare/imaging/dicomadapter/Flags.java)

## Deployment using Kubernetes

The adapters can be deployed to Google Cloud Platform using [GKE] (https://cloud.google.com/kubernetes-engine/). We have published prebuilt Docker images for the both adapters to [Google Container Registry](https://cloud.google.com/container-registry/).

- Import Adapter: `gcr.io/cloud-healthcare-containers/dicom-import-adapter`
- Export Adapter: `gcr.io/cloud-healthcare-containers/dicom-export-adapter`

### Requirements

- A [Google Cloud project](https://cloud.google.com).
- Installed [gcloud](https://cloud.google.com/sdk/gcloud/) and [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command line tools.

### Deploying Docker Images to GKE

Create a local file called `dicom_adapter.yaml`. This file will contain the
configuration specifying the number of adapters to deploy, along with their
command line flags.

To deploy an Import Adapter, add the following to `dicom_adapter.yaml`. Modify
the flags for your use case.

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: dicom-adapter
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: dicom-adapter
    spec:
      containers:
        - name: dicom-import-adapter
          image: gcr.io/cloud-healthcare-containers/dicom-import-adapter:latest
          ports:
            - containerPort: 2575
              protocol: TCP
              name: "port"
          command:
            - "/import/bin/import"
            - "--dimse_aet=IMPORTADAPTER"
            - "--dimse_port=2575"
            - "--dicomweb_addr=http://localhost:80"
            - "--dicomweb_stow_path=/studies"
```

If needed, to additionally include an Export Adapter, you can add the to the
containers in `dicom_adapter.yaml`. Modify the flags for your use case.

```yaml
        - name: dicom-export-adapter
          image: gcr.io/cloud-healthcare-containers/dicom-export-adapter:latest
          command:
            - "--peer_dimse_aet=PEERAET"
            - "--peer_dimse_ip=localhost"
            - "--peer_dimse_port=104"
            - "--project_id=myproject"
            - "--subscription_id=mysub"
            - "--dicomweb_addr=https://localhost:80"
            - "--oauth_scopes=https://www.googleapis.com/auth/pubsub"
```

To deploy the configuration to GKE cluster, execute the following:

```shell
gcloud container clusters create dicom-adapter --zone=us-central1-a --scopes https://www.googleapis.com/auth/cloud-healthcare,https://www.googleapis.com/auth/pubsub
kubectl create -f dicom_adapter.yaml
```

If you are deploying an Import Adapter, you can expose the DIMSE port internally
(e.g. 2575 here). This can be done through a load
balancer. Create a `dicom_adapter_load_balancer.yaml`, and add the following:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: dicom-adapter-load-balancer
  annotations:
    cloud.google.com/load-balancer-type: "Internal"
spec:
  ports:
  - port: 2575
    targetPort: 2575
    protocol: TCP
    name: port
  selector:
    app: dicom-adapter
  type: LoadBalancer
```

To deploy the load balancer, execute the following:

```shell
kubectl create -f dicom_adapter_load_balancer.yaml
```

The status and IP address of load balancer can be seen by executing:

```shell
kubectl get service dicom-adapter-load-balancer
```

## Building from source

As an alternative to using the prebuilt Docker images, you can build the adapters from source code. Both adapters exist as separate binaries and are built using [Gradle](https://gradle.org/). Please refer to these [instructions](https://gradle.org/install/) to build Gradle for your system.

For example, to build Import Adapter:

```shell
cd import
gradle build
```

For example, to additionally execute Import Adapter locally:

```shell
gradle run -Dexec.args="--dimse_aet=IMPORTADAPTER --dimse_port=4008 --dicomweb_addr=http://localhost:80 --dicomweb_stow_path=/studies"
```

### Building and publishing Docker Images

To build and upload Import Adapter Docker images:

```shell
cd import
PROJECT=<Your Google Cloud Project>
TAG=gcr.io/${PROJECT}/dicom-import-adapter
gradle dockerBuildImage -Pdocker_tag=${TAG}
docker push ${TAG}
```

To build and upload Export Adapter Docker images:

```shell
cd export
PROJECT=<Your Google Cloud Project>
TAG=gcr.io/${PROJECT}/dicom-export-adapter
gradle dockerBuildImage -Pdocker_tag=${TAG}
docker push ${TAG}
```
