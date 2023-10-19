# syntax=docker/dockerfile:1

FROM golang:1.21.3-alpine3.18 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && \
    go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/virtual-kubelet ./cmd/virtual-kubelet/main.go

FROM scratch AS final

COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /etc/ssl/cert.pem /etc/ssl/cert.pem
COPY LICENSE /LICENSE
COPY NOTICE /NOTICE
COPY --from=build --chown=root:root /app/bin/virtual-kubelet /usr/bin/virtual-kubelet

ENTRYPOINT ["/usr/bin/virtual-kubelet"]
