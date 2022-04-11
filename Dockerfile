FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.17-openshift-4.10 AS builder
ARG VERSION="1.17"

WORKDIR /opt/app-root
COPY . .

RUN go build -ldflags "-X main.version=${VERSION}" -mod vendor -o node-observability-agent cmd/node-observability-agent.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5

COPY --from=builder /opt/app-root/node-observability-agent ./

ENTRYPOINT ["sh", "-c", "./node-observability-agent --tokenFile /var/run/secrets/kubernetes.io/serviceaccount/token --caCertFile /var/run/secrets/kubernetes.io/serviceaccount/kubelet-serving-ca.crt --storage /host/tmp/pprofs/ --node $NODE_IP"]
