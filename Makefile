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
LOG_LEVEL ?= INFO

# These must be defined in the environmnet before executing 'make install'
SCE_API_KEY ?= api-key
SCE_ORGANIZATION_NAME ?= org-name
SCE_PROJECT_NAME ?= project-name

# Development targets

CMDS := bin/virtual-kubelet

.PHONY: build
build: $(CMDS)

.PHONY: build-image
build-image:
	docker build --tag ghcr.io/saladtechnologies/virtual-kubelet-saladcloud:$(IMAGE_TAG) --file docker/Dockerfile .

.PHONY: clean
clean:
	rm $(CMDS)
	go clean

.PHONY: lint
lint:
	golangci-lint run ./... $(LINT_ARGS)

.PHONY: test
test:
	go test -v ./...

tidy:
	go mod tidy

# The conventional BUILD_VERSION is not very useful at the moment since we are not tagging the repo
# Use the sha for the build as a version for now.
# bin/virtual-kubelet: BUILD_VERSION ?= $(shell git describe --tags --always --dirty="-dev")
bin/virtual-kubelet: BUILD_VERSION ?= $(shell git rev-parse --short HEAD)

# It seems more useful to have the commit date than the build date for ordering versions
# since commit shas have no order
# bin/virtual-kubelet: BUILD_DATE    ?= $(shell date -u '+%Y-%m-%d-%H:%M UTC')
bin/virtual-kubelet: BUILD_DATE    ?= $(shell git log -1 --format=%cd --date=format:"%Y%m%d")

bin/virtual-kubelet: VERSION_FLAGS := -ldflags='-X "main.buildVersion=$(BUILD_VERSION)" -X "main.buildTime=$(BUILD_DATE)"'

bin/%: CGO_ENABLED=0
bin/%:
	go build -ldflags '-extldflags "-static"' -o bin/$(*) $(VERSION_FLAGS) ./cmd/$(*)

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
	  --set provider.logLevel=$(LOG_LEVEL) \
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
