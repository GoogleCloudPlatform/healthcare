# MLLP Adapter

The MLLP(Short for "Minimal Lower Layer Protocol") adapter is a component that runs on [GKE](https://cloud.google.com/kubernetes-engine/), receives HL7 messages via MLLP/TCP, and forwards messages received to HL7 API.

## Requirements

* A [Google Cloud project](https://cloud.google.com).
* A [Docker](https://docs.docker.com/) repository. The following instructions assume the use of [Google Container Registry](https://cloud.google.com/container-registry/).
* Installed [gcloud](https://cloud.google.com/sdk/gcloud/) and [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) command line tools.

## Build

We use bazel as the build tool. Please refer to [the bazel documentation](https://docs.bazel.build/versions/master/getting-started.html) to get started.

Run the following commands to build the MLLP adapter binary:

```bash
bazel build :mllp_adapter
```

Or to build and load a docker image that contains the binary:

```bash
bazel run :mllp_adapter_docker
```

If docker is installed on your system, you should be able to see the image:

```bash
docker images
```

To run the image locally:

```bash
docker run --network=host -v ~/.config:/root/.config bazel:mllp_adapter_docker /usr/mllp_adapter/mllp_adapter --hl7_project_id=<PROJECT_ID> --hl7_location_id=<LOCATION_ID> --hl7_dataset_id=<DATASET_ID> --hl7_store_id=<STORE_ID> --export_stats=false --receiver_ip=127.0.0.1 --pubsub_project_id=<PUBSUB_PROJECT_ID> --pubsub_subscription=<PUBSUB_SUBSCRIPTION_ID> --api_addr_prefix=<API_ADDR_PREFIX>
```

In the command above:
* `--network=host` is used to expose the port of the container;
* `-v ~/.config:/root/.config` is used to give the container access to gcloud credentials;

Also note that:
* `PUBSUB_PROJECT_ID` and `PUBSUB_SUBSCRIPTION_ID` are available by creating a pubsub topic and a subscription on Google Cloud;
* `API_ADDR_PREFIX` is of form `https://www.google.com:443/v1`, scheme, port and version should all be presented.

You should be able to send HL7 messages now:

```bash
# This will fail because the format is invalid.
echo -n -e '\x0btestmessage\x1c\x0d' | telnet localhost 2575
```

## Deployment

Before deploying the docker image to GKE you need to publish the image to a registry.  First modify `BUILD.bazel` to replace `my-project` and `my-image` with real values, then run:

```bash
bazel run :image_push
```

Next create a resource config file `mllp_adapter.yaml` locally. You can use the following as a template but replace the placeholders for your use case.

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: mllp-adapter-deployment
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: mllp-adapter
    spec:
      containers:
        - name: mllp-adapter
          imagePullPolicy: Always
          image: gcr.io/<GCR_PROJECT_ID>/<IMAGE_NAME>:<IMAGE_LABEL>
          ports:
            - containerPort: 2575
              protocol: TCP
              name: "port"
          command:
            - "/usr/mllp_adapter/mllp_adapter"
            - "--port=2575"
            - "--hl7_project_id=<PROJECT_ID>"
            - "--hl7_location_id=<LOCATION_ID>"
            - "--hl7_dataset_id=<DATASET_ID>"
            - "--hl7_store_id=<STORE_ID>"
            - "--api_addr_prefix=<API_ADDR_PREFIX>"
            - "--logtostderr"
            - "--receiver_ip=<RECEIVER_IP>"
            - "--pubsub_project_id=<PUBSUB_PROJECT_ID>"
            - "--pubsub_subscription=<PUBSUB_SUBSCRIPTION_ID>"
```

In addition we need a service `mllp_adapter_service.yaml` to do load balancing:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mllp-adapter-service
  annotations:
    cloud.google.com/load-balancer-type: "Internal"
spec:
  ports:
  - port: 2575
    targetPort: 2575
    protocol: TCP
    name: port
  selector:
    app: mllp-adapter
  type: LoadBalancer
```

Deploy to a GKE cluster:

```bash
gcloud container clusters create mllp-adapter --zone=<ZONE_ID> --scopes https://www.googleapis.com/auth/pubsub
kubectl create -f mllp_adapter.yaml
kubectl create -f mllp_adapter_service.yaml
```

## VPN

*Use E2E VPN setup if want your data to be encrypted end-to-end. See [the "encryption in transit" doc](https://cloud.google.com/security/encryption-in-transit/) for more details.*

### Cloud VPN

[Cloud VPN](https://cloud.google.com/vpn/docs/) creates a secure tunnel to ensure data is encrypted in transit.

First create a static IP address for the VPN gateway (GATEWAY_IP):

```bash
gcloud --project <PROJECT_ID> compute addresses create <IP_NAME> --region=<REGION_ID>
```

Then set up the VPN:

```bash
gcloud --project <PROJECT_ID> compute target-vpn-gateways create mllp-vpn --region <REGION_ID> --network <VPC_NAME>
gcloud --project <PROJECT_ID> compute forwarding-rules create "mllp-vpn-rule-esp" --region <REGION_ID> --address <GATEWAY_IP> --ip-protocol "ESP" --target-vpn-gateway "mllp-vpn"
gcloud --project <PROJECT_ID> compute forwarding-rules create "mllp-vpn-rule-udp500" --region <REGION_ID> --address <GATEWAY_IP> --ip-protocol "UDP" --ports "500" --target-vpn-gateway "mllp-vpn"
gcloud --project <PROJECT_ID> compute forwarding-rules create "mllp-vpn-rule-udp4500" --region <REGION_ID> --address <GATEWAY_IP> --ip-protocol "UDP" --ports "4500" --target-vpn-gateway "mllp-vpn"
```

Next set up the tunnels (also in the opposite direction which isn't shown here):

```bash
gcloud --project <PROJECT_ID>  compute vpn-tunnels create "tunnel-1" --region <REGION_ID>  --peer-address <PEER_IP_ADDRESS> --shared-secret <SHARED_SECRET> --ike-version "2" --target-vpn-gateway "mllp-vpn" --local-traffic-selector="10.100.0.0/24"
gcloud --project <PROJECT_ID>  compute routes create "tunnel-1-route-1" --network <VPC_NAME> --next-hop-vpn-tunnel "tunnel-1" --next-hop-vpn-tunnel-region <REGION_ID> --destination-range "10.101.0.0/24"
```

Finally configure the firewall to allow traffic if necessary:

```bash
gcloud --project <PROJECT_ID> compute firewall-rules create allow-mllp-over-vpn --network <VPC_NAME> --allow tcp:32577,icmp --source-ranges "10.101.0.0/24"
```

### E2E VPN

The docker VPN image used in this section is [here on github](https://github.com/kitten/docker-strongswan). First check the configurations in the repo to see
if you need to make any changes for your use case (you probabaly do).

Pull the image from docker hub and upload it to gcr.io:

```bash
# Build the image from source code instead of pulling it if you made changes in previous step.
docker pull philplckthun/strongswan
docker tag philplckthun/strongswan:latest gcr.io/<GCR_PROJECT_ID>/<IMAGE_NAME>:<IMAGE_LABEL>
gcloud docker -- push gcr.io/<GCR_PROJECT_ID>/<IMAGE_NAME>:<IMAGE_LABEL>
```

Update the resource file to add VPN as a container in the same pod:

```yaml
name: mllp-adapter-deployment
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: mllp-adapter
    spec:
      containers:
        - name: mllp-adapter
          imagePullPolicy: Always
          image: gcr.io/<GCR_PROJECT_ID>/<IMAGE_NAME>:<IMAGE_LABEL>
          ports:
            - containerPort: 2575
              protocol: TCP
              name: "port"
          command:
            - "/usr/mllp_adapter/mllp_adapter"
            - "--port=2575"
            - "--hl7_project_id=<PROJECT_ID>"
            - "--hl7_location_id=<LOCATION_ID>"
            - "--hl7_dataset_id=<DATASET_ID>"
            - "--hl7_store_id=<STORE_ID>"
            - "--api_addr_prefix=<API_ADDR_PREFIX>"
            - "--logtostderr"
            - "--receiver_ip=<RECEIVER_IP>"
            - "--pubsub_project_id=<PUBSUB_PROJECT_ID>"
            - "--pubsub_subscription=<PUBSUB_SUBSCRIPTION_ID>"
        - name: vpn-server
          securityContext:
            privileged: true
          imagePullPolicy: Always
          image: gcr.io/<GCR_PROJECT_ID>/<IMAGE_NAME>:<IMAGE_LABEL>
          ports:
            - name: "ike"
              containerPort: 500
              protocol: UDP
            - name: "natt"
              containerPort: 4500
              protocol: UDP
            - name: "port"
              containerPort: 1701
              protocol: UDP
          volumeMounts:
            - name: modules
              mountPath: /lib/modules
              readOnly: true
      volumes:
      - name: modules
        hostPath:
          path: /lib/modules
          type: Directory
```

Load necessary kernel modules before applying the configuration changes. Then apply the changes:

```bash
kubectl apply -f mllp_adapter.yaml
```
(Optional) Allocate a static IP address for the load balancer:

```bash
gcloud --project <PROJECT_ID> compute addresses create vpn-load-balancer --region=us-central1
```

Update the service config to expose it as an external load balancer:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mllp-adapter-service
spec:
  ports:
  - port: 500
    targetPort: 500
    protocol: UDP
    name: ike
  - port: 4500
    targetPort: 4500
    protocol: UDP
    name: natt
  - port: 1701
    targetPort: 1701
    protocol: UDP
    name: port
  loadBalancerIP: <LOAD_BALANCER_IP>
  selector:
    app: mllp-adapter
  type: LoadBalancer
```

Apply the change as well by running `kubectl apply -f mllp_adapter_service.yaml`

This will also update the firewall rules automatically.

Now connect your client to the VPN server and test if you can access the adapter:

```bash
kubectl describe pods | grep IP: # Get pod IP.
echo -n -e '\x0btestmessage\x1c\x0d' | telnet <POD_IP> 2575
```

## Debug

To view the running status and logs of the pod:

```bash
kubectl get pods
kubectl logs <POD_ID>
```
