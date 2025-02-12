# Virtual Kubelet SaladCloud Walkthrough

*[Note:  This document is a work-in-progress and is lacking certain content, such as auth for a real K8s cluster is not called out yet.*

The virtual-kubelet project enables external API-controlled resources to be made available to a Kubernetes cluster by implementing the Kubelet API.

This document steps through using Salad’s virtual-kubelet-saladcloud (https://github.com/SaladTechnologies/virtual-kubelet-saladcloud) (SCVK) project to connect to Salad Compute Engine (SCE).  It will enable control of SCE Container Groups from the K8s control plane.

### Notes

- K8s uses ‘replica’ to refer to the number of workload instances it manages. This maps to an SCE Container Group. SCE uses ‘replica’ to refer to the number of container instances that are executed in parallel inside a Container Group.  The mapping is at a different layer, setting the K8s replica to 3 will result in 3 Container Groups being launched with the same image, not 3 replicas in a single Container Group.
- The external cloud resource that virtual-kubelet connects to is called the ‘provider’.

## Prerequisites

- Running Kubernetes cluster (can be the Docker Desktop insta-K8s), 1.26+
- Helm v3+
- Go 1.20+

## Build & install virtual-kubelet-saladcloud

SCE’s virtual-kubelet-saladcloud (SCVK) is distributed as source code, you can get the repo from GitHub and build a container image directly:

```bash
git clone [https://github.com/SaladTechnologies/virtual-kubelet-saladcloud](https://github.com/SaladTechnologies/virtual-kubelet-saladcloud/tree/main/charts/virtual-kubelet)
cd [virtual-kubelet-saladcloud](https://github.com/SaladTechnologies/virtual-kubelet-saladcloud/tree/main/charts/virtual-kubelet)
make build-image
```

For development you can build it without a container and run in the foreground for testing.

Normally you will be running the SCVK image in the K8s cluster it is serving but this is not a requirement.  It need access to the K8s control plane network to listen for events and to the provider API endpoint.

## VK Configuration Considerations

The plan is to run a single SCVK that uses SCE as its provider.  The namespace that the VK runs in is not critical, your usual policies for determining that. Probably a good idea to use a separate namespace, we’ll use “sce” here.

Credentials are normally required for the backend provider connection.  For SaladCloud that is an API Key, an Organization Name and a Project Name.  As SCVK is completely configured via command-line options the API key will also be passed in this manner.

- SCE creds
    - API Key - an API key is specific to an SCE user account and may be configured for multiple organizations
    - Organization Name
    - Project Name - an SCVK instance only uses a single provider, available in the Portal web UI

The SCVK Github repo contains a deployment Helm chart that has a number of configurable items, normally only a few are used:

- provider.nodename
- name

SCVK hard-codes a primary taint key: [virtual-kubelet.io/provider](http://virtual-kubelet.io/provider)=saladcloud.  This can be augmented by adding taints manually, or preferably modifying SCVK to accept other key=value pairs.

## Running VK

Once the final configuration is determined SCVK can be deployed by the included Helm chart:

```bash
helm install \
	--create-namespace --namespace sce \
	--set salad.organizationName=salad \
	--set salad.projectName=dtdemo \
	--set salad.apiKey=api-key-value-goes-here \
	--set provider.image.tag=latest \
	--set provider.nodename=sce-vk \
	sce \
	./charts/virtual-kubelet
```

This will create a pod looking something like this:

```bash
$ kubectl --namespace sce get pod
NAME                                                       READY   STATUS    RESTARTS   AGE
sce-virtual-kubelet-saladcloud-provider-67476b4dd5-r5pf8   1/1     Running   0          30m
```

And a new kubelet:

```bash
$ kubectl get node
NAME             STATUS     ROLES           AGE    VERSION
docker-desktop   Ready      control-plane   170m   v1.28.2
sce-vk-yp0       Ready      agent           30m
```

## Running Jobs

Scheduling a pod to SCVK requires some additional configuration in the spec, we need to specify the tolerations that will allow the scheduler to select SCVK and we need to add some Salad-specific configuration.

**Kubetnetes Node Selection**

Setting nodeSelection to virtual-kubelet

```yaml
spec:
  template:
    spec:
      nodeSelector:
        kubernetes.io/role: agent
        type: virtual-kubelet
```

**Taint and Toleration**

SCVK hard-codes the taint value to [virtual-kubelet.io/provider](http://virtual-kubelet.io/provider)=saladcloud.

The toleration thus looks like:

```yaml
spec:
  template:
    spec:
      tolerations:
        - key: virtual-kubelet.io/provider
          operator: Equal
          value: saladcloud
          effect: NoSchedule
```

**Salad Cloud Specifics**

SCE requires a number of configuration items for the Container Group.  These are passed via Salad annotations

[ToDo: find the list of supported annotations]

```yaml
spec:
  template:
    metadata:
      annotations:
        salad.com/country-codes: us
        salad.com/networking-protocol: "http"
        salad.com/networking-port: "1234"
        salad.com/networking-auth: "false"
        salad.com/gpu-classes: "dec851b7-eba7-4457-a319-a01b611a810e"
```

See below for how to get the GPU class UUIDs

### Networking

### Annotations

**GPU Classes**

SCE defines a number of GPU classes that describe the available GPU classes.  The SCE API unfortuantely only accepts UUIDs for those values and the portal web UI does not display these.  Retrieve the full list directly from the API using curl:

```yaml
curl --location 'https://api.salad.com/api/public/organizations/<organization-name>/gpu-classes' \
--header 'Salad-Api-Key: <API Key>'
```

Interestingly the list is filtered by Organization Name.  Don’t forget to URLEncode the org-name if it has spaces or other illegal URL characters in it.

### Health Probes

SCE health probes closely mimic K8s health probes

**Startup Probe**

The following probe configurations in a K8s spec:

```yaml
spec:
  template:
    spec:
      containers:
        - image: XXXXX
          livenessProbe:
            tcpSocket:
              port: 80
            initialDelaySeconds: 10
            failureThreshold: 2
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /
              port: 80
            initialDelaySeconds: 30
            failureThreshold: 2
            periodSeconds: 10
          startupProbe:
            tcpSocket:
              port: 80
            initialDelaySeconds: 20
            failureThreshold: 90
            periodSeconds: 10
```

creates this in SCE:

```yaml
            "liveness_probe": {
                "tcp": {
                    "port": 80
                },
                "initial_delay_seconds": 10,
                "period_seconds": 10,
                "timeout_seconds": 1,
                "success_threshold": 1,
                "failure_threshold": 2
            },
            "readiness_probe": {
                "http": {
                    "path": "/",
                    "port": 80,
                    "scheme": 0,
                    "headers": []
                },
                "initial_delay_seconds": 30,
                "period_seconds": 10,
                "timeout_seconds": 1,
                "success_threshold": 1,
                "failure_threshold": 2
            },
            "startup_probe": {
                "tcp": {
                    "port": 80
                },
                "initial_delay_seconds": 20,
                "period_seconds": 10,
                "timeout_seconds": 1,
                "success_threshold": 1,
                "failure_threshold": 90
            },
```

### Additional Notes

- Use SI units (power-of-ten) for RAM in the container spec rather than the usual power-of-two units. SCE will bump up to the next GB size, ie “2G” gets a 2GB instance, “2Gi” gets a 3GB instance.
