FROM golang:1.21 as builder
ARG VERSION="1.21"

WORKDIR /opt/app-root
COPY . .

RUN go build -ldflags "-X main.version=${VERSION}" -mod vendor -o node-observability-agent cmd/node-observability-agent/main.go

FROM registry.access.redhat.com/ubi9/ubi-init:latest

RUN mkdir /run/node-observability && \
    chgrp -R 0 /run/node-observability && \
    chmod -R g=u /run/node-observability

COPY --from=builder /opt/app-root/node-observability-agent /usr/bin/

# taken from KCS article https://access.redhat.com/solutions/5343671
RUN dnf install -y tc perf psmisc hostname sysstat iotop conntrack-tools ethtool numactl net-tools

RUN mkdir -p /tmp/scripts && mkdir -p /tmp/results 
COPY scripts/metrics/* /tmp/scripts
COPY ./uid_entrypoint.sh ./uid_entrypoint.sh

EXPOSE 9000

USER 65532:65532

# this allows us to set the command and args in the deploy config
ENTRYPOINT ["./uid_entrypoint.sh"]
