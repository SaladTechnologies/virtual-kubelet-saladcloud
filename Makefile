# virtual-kubelet-saladcloud

# Standard Development Targets
# build - build the project, binaries go into ./bin
# build-image - build a Docker image with the virtual-kubelet binary
# clean - clean up built binaries and cached Go artifacts
# lint - run golangci-lint (set args in LINT_ARGS)
# tidy - "go mod tidy"

# Deployment and Debug Targets
# install - install the Docker image into the current k8s environment
#   (add additional args in INSTALL_ARGS)
# dry-install - run install with --dry-run
# uninstall - delete the Docker image deployment from the current k8s environment
# run - run bin/virtual-kubelet in the foreground using the SCE_* environment variables

IMAGE_TAG ?= latest

NAMESPACE ?= saladcloud
DEPLOYMENT_NAME ?= saladcloud-node

# These must be defined in the environmnet before executing 'make install'
SCE_API_KEY ?= api-key
SCE_ORGANIZATION_NAME ?= org-name
SCE_PROJECT_NAME ?= project-name

# Development targets

.PHONY: build
build: CGO_ENABLED=0
build:
	go build -o ./bin/virtual-kubelet ./cmd/virtual-kubelet/main.go

.PHONY: build-image
build-image:
	docker build --tag ghcr.io/saladtechnologies/virtual-kubelet-saladcloud:$(IMAGE_TAG) --file docker/Dockerfile .

.PHONY: clean
clean:
	rm -rf ./bin
	go clean

.PHONY: lint
lint:
	golangci-lint run ./... $(LINT_ARGS)

.PHONY: test
test:

tidy:
	go mod tidy

# Deploy and debug targets

# Install and start the Docker image in k8s
.PHONY: install
install:
	helm install \
	  --create-namespace \
	  --namespace $(NAMESPACE) \
	  --set salad.apiKey=$(SCE_API_KEY) \
	  --set salad.organizationName=$(SCE_ORGANIZATION_NAME) \
	  --set salad.projectName=$(SCE_PROJECT_NAME) \
	  --set provider.image.tag=$(IMAGE_TAG) \
	  $(INSTALL_ARGS) \
	  $(DEPLOYMENT_NAME) \
	  ./charts/virtual-kubelet

.PHONY: dry-install
dry-install:
	$(MAKE) install INSTALL_ARGS="--dry-run"

.PHONY: uninstall
uninstall:
	helm uninstall \
	  --namespace $(NAMESPACE) \
	  $(DEPLOYMENT_NAME)

# Run in foreground for debugging
.PHONY: run
run:
	bin/virtual-kubelet \
	  --nodename $(DEPLOYMENT_NAME) \
	  --log-level DEBUG \
	  --disable-taint \
	  --sce-organization-name $(SCE_ORGANIZATION_NAME) \
	  --sce-project-name $(SCE_PROJECT_NAME) \
	  --sce-api-key $(SCE_API_KEY)
