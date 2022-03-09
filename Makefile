GO=GO111MODULE=on GOFLAGS=-mod=vendor go
GO_BUILD_BINDIR := bin
GO_TEST_FLAGS :=-timeout=50s -tags=fake
GO_PACKAGES=./pkg/... ./cmd/... ./tools/...

ifeq (,$(shell which podman 2>/dev/null))
CONTAINER_ENGINE ?= docker
else
CONTAINER_ENGINE ?= podman
endif

include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/images.mk \
	targets/openshift/deps-gomod.mk \
)

# Image URL to use all building/pushing image targets
IMG ?= node-observability-agent
IMAGE_TAG ?= $(shell git rev-parse --short HEAD)

.PHONY: build.image
build.image: build.go verify
	$(CONTAINER_ENGINE) build -t ${IMG}:${IMAGE_TAG} .

.PHONY: push.image.rhel8
build.image.rhel8:
	$(CONTAINER_ENGINE) build -t ${IMG}:${IMAGE_TAG} -f Dockerfile.rhel8 .

.PHONY: push.image
push.image: build.image
	$(CONTAINER_ENGINE) push ${IMG}:$(IMAGE_TAG)

.PHONY: push.image.rhel8
push.image.rhel8: build.image.rhel8
	$(CONTAINER_ENGINE) push ${IMG}:${IMAGE_TAG}

deploy: push.image.rhel8
	oc new-project node-observability-operator
	IMG=$(IMG) hack/kustomize-build.sh | oc apply -f -
