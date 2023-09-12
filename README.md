# Virtual Kubelet Provider for SaladCloud

[![License](https://img.shields.io/github/license/SaladTechnologies/virtual-kubelet-saladcloud)](./LICENSE) [![CI Workflow](https://github.com/SaladTechnologies/virtual-kubelet-saladcloud/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/SaladTechnologies/virtual-kubelet-saladcloud/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/SaladTechnologies/virtual-kubelet-saladcloud)](https://goreportcard.com/report/github.com/SaladTechnologies/virtual-kubelet-saladcloud)

Salad's Virtual Kubelet (VK) provider for SaladCloud enables running Kubernetes (K8s) pods as container group deployments.

To Setup the project
1. Clone the repo command
```bash
git clone
```
2. Install the dependencies
```bash
go mod download
```
3. Build the project
```bash
go build
```
4. Run the project
```bash
go run main.go --nodename {valid_node_name} --projectName {projectName} --organizationName {organizationName} --api-key {api-key} --kubeconfig {kubeconfig}
```

## Prerequisites
You should have valid configuration required to run the project

go to portal.salad.io and create a project and get the api-key and organization name
set the kubeconfig to the valid kubeconfig file


1. Valid ApiKey and OrganizationName
2. Valid Kubeconfig
3. Valid NodeName
4. Valid ProjectName

