FROM golang:1.17 as builder
ARG VERSION="1.17"

WORKDIR /opt/app-root
COPY . .

RUN go build -ldflags "-X main.version=${VERSION}" -mod vendor -o node-observability-agent cmd/node-observability-agent/main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5

COPY --from=builder /opt/app-root/node-observability-agent /usr/bin/
USER 65532:65532
#TODO(alebedev): SET UP THE ENTRYPOINT TO THE OPERATOR BINARY!
