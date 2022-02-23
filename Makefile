# Image URL to use all building/pushing image targets
IMG ?= node-observability-agent:latest
CONTAINER_ENGINE ?= podman

all: build.image push.image
build.image:
	$(CONTAINER_ENGINE) build -t ${IMG} .
push.image:
	$(CONTAINER_ENGINE) tag ${IMG} quay.io/skhoury/${IMG}
	$(CONTAINER_ENGINE) push quay.io/skhoury/${IMG}