# syntax=docker/dockerfile:1

FROM golang:1.24.2-alpine3.21 AS build
WORKDIR /app
ARG VERSION_FLAGS=-ldflags=

COPY go.mod go.sum ./
RUN go mod download && \
    go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build "${VERSION_FLAGS}" -o ./bin/virtual-kubelet-saladcloud ./cmd/virtual-kubelet-saladcloud/main.go

FROM scratch AS final

COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY LICENSE /LICENSE
COPY NOTICE /NOTICE
COPY --from=build --chown=root:root /app/bin/virtual-kubelet-saladcloud /usr/bin/virtual-kubelet-saladcloud

ENTRYPOINT ["/usr/bin/virtual-kubelet-saladcloud"]
