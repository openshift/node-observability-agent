FROM golang:1.17 as builder
ARG VERSION="1.17"

WORKDIR /opt/app-root
COPY . .

RUN go build -ldflags "-X main.version=${VERSION}" -mod vendor -o node-observability-agent cmd/node-observability-agent/main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5

RUN chgrp 0 /run && \
    chmod g=u /run && \
    mkdir /run/node-observability && \
    chgrp -R 0 /run/node-observability && \
    chmod -R g=u /run/node-observability

COPY --from=builder /opt/app-root/node-observability-agent /usr/bin/

ENTRYPOINT ["sh", "-c", "node-observability-agent --tokenFile /var/run/secrets/kubernetes.io/serviceaccount/token --storage /host/tmp/pprofs/"]
