# SaladCloud Virtual Kubelet provider

## Installation

Quick start instructions for the setup using Helm.

### Prerequisites

- [Helm](https://helm.sh/docs/intro/quickstart/#install-helm)
- [Kubernetes 1.26+](https://kubernetes.io/docs/setup/)

### Installing the chart

1. Clone project

   ```sh
   git clone https://github.com/SaladTechnologies/virtual-kubelet-saladcloud.git
   cd charts/virtual-kubelet
   ```

2. Install using Helm v3.0+

   ```shell
   helm install \
     --create-namespace \
     --namespace saladcloud
     --set salad.apiKey=$SCE_API_KEY \
     --set salad.organizationName=$SCE_ORGANIZATION_NAME \
     --set salad.projectName=$SCE_PROJECT_NAME \
     saladcloud .
   ```

3. Verify that the pod is running and the virtual node available

   ```sh
   kubectl get nodes
   ```

   <details>
   <summary>Result</summary>

   ```shell
   NAME                                   STATUS    ROLES     AGE       VERSION
   saladcloud-node                        Ready     agent     2m        v1.0.0
   ```

   </details>
