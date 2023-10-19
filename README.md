# SaladCloud Virtual Kubelet Provider

[![License](https://img.shields.io/github/license/SaladTechnologies/virtual-kubelet-saladcloud)](./LICENSE) [![CI Workflow](https://github.com/SaladTechnologies/virtual-kubelet-saladcloud/actions/workflows/ci.yml/badge.svg?branch=main&event=push)](https://github.com/SaladTechnologies/virtual-kubelet-saladcloud/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/SaladTechnologies/virtual-kubelet-saladcloud)](https://goreportcard.com/report/github.com/SaladTechnologies/virtual-kubelet-saladcloud)

Salad's Virtual Kubelet (VK) provider for SaladCloud enables running Kubernetes (K8s) pods as container group deployments.

## Development

Follow the steps below to get started with local development.

### Prerequisites

- [Git](https://git-scm.com/downloads)
- [Go](https://go.dev/dl)
- [Docker Desktop](https://docs.docker.com/get-docker/) with a [local Kubernetes cluster](https://docs.docker.com/desktop/kubernetes/)

### Getting Started

1. Clone the repository.

   ```sh
   git clone https://github.com/SaladTechnologies/virtual-kubelet-saladcloud.git
   ```

2. Restore the dependencies.

   ```sh
   go mod download
   go mod verify
   ```

3. Build the project.

   ```sh
   go build -o ./bin/virtual-kubelet ./cmd/virtual-kubelet/main.go
   ```

4. Run the project.

   ```sh
   ./bin/virtual-kubelet --sce-api-key {apiKey} --sce-project-name {projectName} --sce-organization-name {organizationName}
   ```
