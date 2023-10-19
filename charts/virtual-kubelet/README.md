# SaladCloud Virtual Kubelet Provider



![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)  ![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) 

Deploy containers to SaladCloud from your Kubernetes cluster using a virtual node.

## Installation

Follow the steps below to get started with the SaladCloud Virtual Kubelet Provider.

### Prerequisites

- [Kubernetes 1.26+](https://kubernetes.io/docs/setup/)
- [Helm v3+](https://helm.sh/docs/intro/quickstart/#install-helm)

### Installing the chart

1. Clone the repository.

   ```sh
   git clone https://github.com/SaladTechnologies/virtual-kubelet-saladcloud.git
   ```

2. Install the chart.

   ```shell
   helm install \
     --create-namespace \
     --namespace saladcloud
     --set salad.apiKey=$SCE_API_KEY \
     --set salad.organizationName=$SCE_ORGANIZATION_NAME \
     --set salad.projectName=$SCE_PROJECT_NAME \
     mysaladcloud ./charts/virtual-kubelet
   ```

3. Verify that the virtual node is available.

   ```sh
   kubectl get nodes
   ```

   <details>
   <summary>Results</summary>

   ```shell
   NAME                                   STATUS    ROLES     AGE       VERSION
   saladcloud-node                        Ready     agent     1m        v1.0.0
   ```

   </details>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| additionalLabels | object | `{}` | The collection of additional labels applied to all resources. |
| clusterRoleNameOverride | string | `""` | When set to a non-empty value, overrides the `virtual-kubelet-saladcloud.fullName` value in the `ClusterRoleBinding` resource. |
| fullNameOverride | string | `""` | When set to a non-empty value, overrides the `virtual-kubelet-saladcloud.fullName` value. |
| imagePullSecrets | list | `[]` | The list of `Secret` names containing the registry credentials. |
| name | string | `"virtual-kubelet"` | The SaladCloud Virtual Kubelet Provider name. |
| nameOverride | string | `""` | When set to a non-empty value, overrides the `virtual-kubelet-saladcloud.name` value. |
| provider.deploymentAnnotations | object | `{}` | The collection of annotations applied to the `Deployment` resource. |
| provider.image.digest | string | `""` | The SaladCloud Virtual Kubelet Provider image digest. When set to a non-empty value, the image is pulled by digest (ignore the tag value). |
| provider.image.pullPolicy | string | `"IfNotPresent"` | The `imagePullPolicy` for the SaladCloud Virtual Kubelet Provider image. May be `IfNotPresent`, `Always`, or `Never`. |
| provider.image.repository | string | `"ghcr.io/saladtechnologies/virtual-kubelet-saladcloud"` | The SaladCloud Virtual Kubelet Provider image repository URI. |
| provider.image.tag | string | `""` | The SaladCloud Virtual Kubelet Provider image tag. |
| provider.logLevel | string | `"warn"` | The log level. May be `error`, `warn`, `info`, `debug`, or `trace`. |
| provider.nodeSelector | object | `{}` | The collection of labels used to assign the SaladCloud Virtual Kubelet Provider pod. |
| provider.nodename | string | `"saladcloud-node"` | The SaladCloud Virtual Kubelet Provider node name. |
| provider.podAnnotations | object | `{}` | The collection of annotations applied to the `Pod` resources created by the deployment. |
| provider.taintEnabled | bool | `true` | The flag indicating whether the SaladCloud Virtual Kubelet Provider node is tainted. |
| rbac.annotations | object | `{}` | The collection of annotations applied to the `ClusterRoleBinding` resource. |
| rbac.clusterRoleName | string | `"cluster-admin"` | The name of the `ClusterRole` resource. |
| rbac.create | bool | `true` | The flag indicating whether the `ClusterRoleBinding` resource should be created. |
| salad.apiKey | string | `""` | The SaladCloud API key. |
| salad.organizationName | string | `""` | The SaladCloud organization name. |
| salad.projectName | string | `""` | The SaladCloud project name. |
| serviceAccount.annotations | object | `{}` | The collection of annotations applied to the `ServiceAccount` resource. |
| serviceAccount.create | bool | `true` | The flag indicating whether the `ServiceAccount` resources should be created. |
| serviceAccountNameOverride | string | `""` | When set to a non-empty value, overrides the `virtual-kubelet-saladcloud.fullName` value in the `ServiceAccount` resource. |

## Requirements

Kubernetes: `>=v1.26.0-0`



## Source Code

* <https://github.com/SaladTechnologies/virtual-kubelet-saladcloud>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Salad Chefs | <dev@salad.com> | <https://salad.com/> |
