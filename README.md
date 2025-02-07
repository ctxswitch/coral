# Coral

Coral is the start of a set of services for kubernetes that provides a structural framework for running applications.  The first iteration provides image and artifact management tools which lets users:
1. Prefetch external container images to kubernetes nodes or to internal local registries.  It allows the user to selectively target groups of nodes for images to be fetched on, and tracks image availability on the nodes.  This allows the service to provide a mutating admissions webhook which can optionally modify the node selectors in resource specifications to ensure applications are only started on nodes that have the image available.  The webhook can also optionally modify the pull policies for containers to ensure that containers do not pull external containers and only rely on the configured container images that are already present in the cluster.
2. Introduces versioned artifact synchronization to nodes.  I initially used this in a (private) flink operator implementation where large jar archives were mounted into the pods by an admissions webhook which set up the volumes and mounts.  This allowed application container to remain small at the expense of additional storage on the nodes.

Over time, we expect to add additional features to Coral including:
- [ ] Fully managed internal registries.
- [ ] A build system (most likely leveraging kaniko) that builds and stores to internal registries allowing users to bypass external registries all together.
- [ ] Registry mirroring and cross-cluster image synchronization.
- [ ] Additional garbage collection and lifecycle management features for container images on both the nodes and internal registries.
 
## Installation

TODO

## Usage

### Prefetching images from external repositories directly to the nodes.

TODO

### Mirroring images from external repositories to an internal repository.

TODO

### Inject pull policies and selectors

Coral gives the option of modifying the pull policies and node selectors of any managed resource.  This allows the user to restrict pods to nodes that already have the image present and also ensure that the pod does not try and pull images externally.  You can control this through annotations on the resource.

The default injection policy is to not modify any resources.  However, the webhook is set up by default to handle and check all resources in all namespaces other than ones that match `kube-*`, `*-system`, `coral`, and `cert-manager`.  Access is configurable through the webhook configuration.  There are several examples listed in [the injection webook configuration](docs/injection-webhook-config.md) document.

#### Pull policy

Coral can ensure that the pull policy for all containers in the resource are set to `never` thereby restricting resources to images already present on the nodes.  To have coral override pull policies for a resource, add the following annotation:

```
image.coral.stvz.io/inject: pull-policy
```

You can exclude containers from the injection policies by using the following annotation with a comma seperated list of container names:

```
image.coral.stvz.io/exclude: [container-name],...
```

The alternative to exclude is the include annotation which will only include containers listed for policy injection:

```
image.coral.stvz.io/include: [container-name],...
```

If both the `include` and `exclude` annotations are present, `exclude` will be ignored.

#### Node selection

Node selection can be used to ensure that pods are started up on nodes that already have the image fetched.  Coral fetch workers will label the nodes once a managed image is present on the node allowing us to gate scheduling on that node to ensure there are no disruptions and to minimize startup latency.  To enable the injection of node selectors into your resources, enable the following:

```
image.coral.stvz.io/inject: selectors
```

By default, all container images are included as part of the selector. As with the policy injection, containers can be included and excluded from:

```
image.coral.stvz.io/include: [container-name],...
```

and 

```
image.coral.stvz.io/exclude: [container-name],...
```

#### Multiple injectors

Multiple injection rules can be specified by including a comma seperated list for the annotation value:

```
image.coral.stvz.io/inject: pull-policy,selectors
```

#### Resources that are supported for injection

##### Core (v1)
* Pods
* ReplicationController

##### Apps (v1)
* DaemonSets
* Deployments
* ReplicaSets
* ReplicationController
* StatefulSets

##### Batch (v1)
* CronJobs
* Jobs

TODO

### Enabling pull policy mutations

TODO

## Fetch workers

TODO

### Configuration

TODO

### Security concerns

The fetch workers interact with the node by mounting the runtime socket and using the Kubernetes CRI-API wrapper around the container runtime environment.  This does introduce potential attack vectors to the service and is generally discouraged.  With this in mind, we built the service to minimize the surface area exposed.

1) The fetch worker containers are built without an operating system or system utilities which do not provide any way to execute commands remotely.  We are only interacting with images through the Kubernetes CRI-API and not fetching images directly.
2) Exposed APIs are read only (currently only exposing metrics).

This should minimize the potential for abuse considerably.

## Potential issues

* Kubernetes provides internal image [https://kubernetes.io/docs/concepts/architecture/garbage-collection/#container-image-garbage-collection](garbage collection based on a series of constraints).  The fetch workers rely on both node labels that it manages and the availability of the image as reported by the node to determine whether or not to fetch it or where it is in it's lifecycle.  If the image has been expunged from the node, the fetch client will attempt to retrieve it again potentially leading to thrashing if the GC is caused by a disk pressure situation.  One possible solution would be to disable the kubelet's image garbage collection.  Coral will track disk/pid pressure situations and will not fetch new images until addressed.  This state emits metrics for monitoring and alerting, which will allow for the user to respond and remove images more intellegently. However, this only makes sense if you are managing all images through coral.  There are efforts being planned to handle this situation more gracefully.

## Development

TODO