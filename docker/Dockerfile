# syntax=docker/dockerfile:1

FROM golang:1.21-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/virtual-kubelet ./cmd/virtual-kubelet/main.go

FROM scratch

COPY --from=builder --chown=root:root /app/bin/virtual-kubelet /usr/bin/virtual-kubelet

ENTRYPOINT ["/usr/bin/virtual-kubelet"]
