# Building

Follow the steps below to get started with building and running locally.

## Prerequisites

- [Git](https://git-scm.com/downloads)
- [Go](https://go.dev/dl)
- [Docker Desktop](https://docs.docker.com/get-docker/) with a [local Kubernetes cluster](https://docs.docker.com/desktop/kubernetes/)

## Getting Started

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
   make build && make build-image
   ```

4. Run the project in the foreground:

   ```sh
   ./bin/virtual-kubelet-saladcloud --sce-api-key {apiKey} --sce-project-name {projectName} --sce-organization-name {organizationName}
   ```
