# virtual-kubelet-saladcloud

.PHONY: build
build: CGO_ENABLED=0
build:
	go build -o ./bin/virtual-kubelet ./cmd/virtual-kubelet/main.go

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
