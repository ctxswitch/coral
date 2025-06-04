# Coral

Coral is a set of services for kubernetes that provides a structural framework for running applications.  The first iteration provides image and artifact management tools which lets users:
1. Prefetch external container images to kubernetes nodes. 
2. Prefetch artifacts from http or s3 endpoints to kubernetes nodes and inject host volume mounts into pods.
3. Prefetch external container images to local registries.  Cross cluster image synchronization will also be supported.
4. Build services that can be used to 

## Installation

TODO

## Usage

### Prefetching images from external repositories directly to the nodes.

TODO

### Mirroring images from external repositories to an internal repository.

TODO

### Security concerns

The fetch workers interact with the node by mounting the runtime socket and using the Kubernetes CRI-API wrapper around the container runtime environment.  This does introduce potential attack vectors to the service and is generally discouraged.  With this in mind, we built the service to minimize the surface area exposed.

1) The fetch worker containers are built without an operating system or system utilities which do not provide any way to execute commands remotely.  We are only interacting with images through the Kubernetes CRI-API and not fetching images directly.
2) Exposed APIs are read only (currently only exposing metrics).

This should minimize the potential for abuse considerably.

## Potential issues

* Kubernetes provides internal image [https://kubernetes.io/docs/concepts/architecture/garbage-collection/#container-image-garbage-collection](garbage collection based on a series of constraints). The node agents will make a best effort attempt to keep the images available, but there is no guarantee that the images will be available at all times.  Currently the only time the agent will attempt to fetch a container image is when the imagesync resource is created or updated and if the node is considered in a healthy state.
* Garbage collection for container images on the kubelet is governed by low and high thresholds.  The kubelet deletes images in order based on the last time they were used starting with the oldest first. This will favor recent images which are more likely to be used, but could potentially cause churn.  As of Kubernetes 1.26, the `pod_start_sli_duration_seconds` metric is available to track pod startup latency which will include the image pull time and depending on the service may be a useful way to monitor the impacts of unexpected container image fetches.

## Development

TODO