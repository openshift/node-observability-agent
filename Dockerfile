FROM registry.access.redhat.com/ubi8/go-toolset:1.15.14-14 as builder
ARG VERSION=""

WORKDIR /opt/app-root
COPY . .

RUN go build -ldflags "-X main.version=${VERSION}" -mod vendor -o node-observability-agent cmd/node-observability-agent.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5

COPY --from=builder /opt/app-root/node-observability-agent ./

ENTRYPOINT ["./node-observability-agent", "--tokenFile", "/var/run/secrets/kubernetes.io/serviceaccount/token", "--storage", "/host/tmp/pprofs/"]
