# virtual-kubelet-saladcloud

# Standard Development Targets
# build - build the project, binaries go into ./bin
# build-image - build a Docker image with the virtual-kubelet-saladcloud binary
# clean - clean up built binaries and cached Go artifacts
# lint - run golangci-lint (set args in LINT_ARGS)
# tidy - "go mod tidy"

IMAGE_TAG ?= latest
CMDS := bin/virtual-kubelet-saladcloud


# The conventional BUILD_VERSION is not very useful at the moment since we are not tagging the repo
# Use the sha for the build as a version for now.
# BUILD_VERSION ?= $(shell git describe --tags --always --dirty="-dev")
BUILD_VERSION ?= $(shell git rev-parse --short HEAD)

# It seems more useful to have the commit date than the build date for ordering versions
# since commit shas have no order
# BUILD_DATE ?= $(shell date -u '+%Y-%m-%d-%H:%M UTC')
BUILD_DATE ?= $(shell git log -1 --format=%cd --date=format:"%Y%m%d")

VERSION_FLAGS := -ldflags='-X "main.buildVersion=$(BUILD_VERSION)" -X "main.buildTime=$(BUILD_DATE)"'

.PHONY: build
build: $(CMDS)

.PHONY: build-image
build-image:
	docker build \
		--tag ghcr.io/saladtechnologies/virtual-kubelet-saladcloud:$(IMAGE_TAG) \
		--file docker/Dockerfile \
		--build-arg VERSION_FLAGS=$(VERSION_FLAGS) \
		.

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

bin/virtual-kubelet-saladcloud:

bin/%: CGO_ENABLED=0
bin/%:
	go build -ldflags '-extldflags "-static"' -o bin/$(*) $(VERSION_FLAGS) ./cmd/$(*)
