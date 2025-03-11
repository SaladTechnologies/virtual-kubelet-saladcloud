# SaladCloud Virtual Kubelet



![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)  ![Version: 0.0.0](https://img.shields.io/badge/Version-0.0.0-informational?style=flat-square) 

Deploy containers to SaladCloud from your Kubernetes cluster using a virtual node.

## Getting Started

Install the Helm chart by running the following command:

   ```shell
   helm install \
     --create-namespace \
     --namespace salad-cloud
     --set salad.apiKey=$SALAD_API_KEY \
     --set salad.organizationName=$SALAD_ORGANIZATION_NAME \
     --set salad.projectName=$SALAD_PROJECT_NAME \
     mynode oci://ghcr.io/saladtechnologies/virtual-kubelet-saladcloud-chart --version 1.0.0
   ```

Verify the SaladCloud virtual node is ready by running the following command:

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
| clusterRoleBinding.additionalLabels | object | `{}` | Additional labels to add to the cluster role binding. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for more information about labels. |
| clusterRoleBinding.annotations | object | `{}` | Annotations to add to the cluster role binding. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) for more information about annotations. |
| clusterRoleBinding.create | bool | `true` | Specifies whether a cluster role binding for the service account should be created. |
| clusterRoleBinding.name | string | `""` | The name of the cluster role binding. If undefined or empty, defaults to the fullname template. |
| commonAnnotations | object | `{}` | Common annotations to add to all resources. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) for more information about annotations. |
| commonLabels | object | `{}` | Common labels to add to all resources. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for more information about labels. |
| deployment.additionalLabels | object | `{}` | Additional labels to add to the deployment. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for more information about labels. |
| deployment.annotations | object | `{}` | Annotations to add to the deployment. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) for more information about annotations. |
| deployment.name | string | `""` | The name of the deployment. If undefined or empty, defaults to the fullname template. |
| fullnameOverride | string | `""` | Overrides the fullname template. This is used as the default resource name. When undefined or empty, this defaults to the Helm release name combined with the name template (see `nameOverride`). |
| imageDigest | string | `""` | The image digest for the SaladCloud Virtual Kubelet container that runs on one of your Kubernetes cluster's worker nodes. This may be used in place of `imageTag` to specify a specific image version. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/containers/images/) for more information about images. |
| imageName | string | `"virtual-kubelet-saladcloud"` | The image name for the SaladCloud Virtual Kubelet container that runs on one of your Kubernetes cluster's worker nodes. This is appended to the `imageRegistry`. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/containers/images/) for more information about images. |
| imagePullPolicy | string | `"IfNotPresent"` | The image pull policy for the SaladCloud Virtual Kubelet container that runs on one of your Kubernetes cluster's worker nodes. The default is `IfNotPresent`, which skips pulling the SaladCloud Virtual Kubelet image if it already exists on the Kubernetes cluster's worker node. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy) for more information about the image pull policy. |
| imagePullSecrets | list | `[]` | The image pull secrets for the SaladCloud Virtual Kubelet container that runs on one of your Kubernetes cluster's worker nodes. The default image repository on the GitHub Container Registry is public and does not require a secret. Use this if you override the image repository (`imageRegistry` and `imageName`) and it is private. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/containers/images/#using-a-private-registry) for more information about private images. |
| imageRegistry | string | `"ghcr.io/saladtechnologies"` | The image registry for the SaladCloud Virtual Kubelet container that runs on one of your Kubernetes cluster's worker nodes. This is prepended to the `imageName`. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/containers/images/) for more information about images. |
| imageTag | string | `"main"` | The image tag for the SaladCloud Virtual Kubelet container that runs on one of your Kubernetes cluster's worker nodes. This defaults to the chart's `version`. This is ignored when `imageDigest` is defined and not empty. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/containers/images/) for more information about images. |
| nameOverride | string | `""` | Overrides the name template. This is used as the value of the `app.kubernetes.io/name` annotation and in the fullname template (with the Helm release name). When undefined or empty, this defaults to the value "virtual-kubelet-saladcloud". |
| namespaceOverride | string | `""` | Overrides the namespace. When undefined or empty, this defaults to the target namespace of the Helm release. |
| pod.additionalLabels | object | `{}` | Additional labels to add to the pod. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for more information about labels. |
| pod.affinity | object | `{}` | The affinity constraints to control scheduling the SaladCloud Virtual Kubelet pod on one of your Kubernetes cluster's worker nodes. |
| pod.annotations | object | `{}` | Annotations to add to the pod. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) for more information about annotations. |
| pod.containerSecurityContext | object | `{}` | The security context to apply to the SaladCloud Virtual Kubelet container. |
| pod.nodeSelector | object | `{"kubernetes.io/arch":"amd64","kubernetes.io/os":"linux"}` | The node constraints to control scheduling the SaladCloud Virtual Kubelet pod on one of your Kubernetes cluster's worker nodes. |
| pod.podSecurityContext | object | `{}` | The security context to apply to the SaladCloud Virtual Kubelet pod. |
| pod.priorityClassName | string | `""` | The priority class name of the SaladCloud Virtual Kubelet pod. |
| pod.resources | object | `{}` | The resource constraints of the SaladCloud Virtual Kubelet pod. |
| pod.schedulerName | string | `""` | The name of the Kubernetes scheduler to handle scheduling the SaladCloud Virtual Kubelet pod on one of your Kubernetes cluster's worker nodes. |
| pod.tolerations | list | `[]` | The tolerations to apply to the SaladCloud Virtual Kubelet pod. |
| pod.volumeMounts | list | `[]` | The additional volume mounts to add to the SaladCloud Virtual Kubelet container. |
| pod.volumes | list | `[]` | The additional volumes to add to the SaladCloud Virtual Kubelet pod. |
| salad.apiKey | string | `""` | The SaladCloud API key used by the SaladCloud Virtual Kubelet to manage container group deployments. The API key must be associated with a user account with access to the organization and project (see `salad.organizationName` and `salad.projectName`). |
| salad.debug | bool | `false` | Specifies whether debug logging is enabled in the SaladCloud Virtual Kubelet. |
| salad.nodeName | string | `""` | The Kubernetes node name used by the SaladCloud Virtual Kubelet. |
| salad.organizationName | string | `""` | The SaladCloud organization name used by the SaladCloud Virtual Kubelet to manage container group deployments. |
| salad.projectName | string | `""` | The SaladCloud project name used by the SaladCloud Virtual Kubelet to manage container group deployments. |
| secret.additionalLabels | object | `{}` | Additional labels to add to the secret. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for more information about labels. |
| secret.annotations | object | `{}` | Annotations to add to the secret. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) for more information about annotations. |
| secret.create | bool | `true` | Specifies whether a secret for the SaladCloud API key should be created. |
| secret.name | string | `""` | The name of the secret. If undefined or empty, defaults to the fullname template. |
| serviceAccount.additionalLabels | object | `{}` | Additional labels to add to the service account. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) for more information about labels. |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account. See the [Kubernetes documentation](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) for more information about annotations. |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created. |
| serviceAccount.name | string | `""` | The name of the service account. If undefined or empty and `serviceAccount.create` is `true`, defaults to the fullname template. If undefined or empty and `serviceAccount.create` is `false`, defaults to the "default" service account in the target namespace. |

## Requirements

Kubernetes: `>=v1.29.0-0`



## Source Code

* <https://github.com/SaladTechnologies/virtual-kubelet-saladcloud>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Salad Chefs | <dev@salad.com> | <https://salad.com/> |
