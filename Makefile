GO=GO111MODULE=on GOFLAGS=-mod=vendor go
GO_BUILD_BINDIR := bin
GO_TEST_FLAGS :=-timeout=50s -tags=fake
GO_PACKAGES=./pkg/... ./cmd/... ./tools/...

GOLANGCI_LINT_BIN=$(shell pwd)/bin/golangci-lint

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
IMG ?= node-observability-agent:go-latest
TARGET_REPO ?= registry.ci.openshift.org/ocp
IMAGE_TAG ?= v0.0.1

.PHONY: lint
## Checks the code with golangci-lint
lint: $(GOLANGCI_LINT_BIN)
	$(GOLANGCI_LINT_BIN) run -c .golangci.yaml --deadline=30m

$(GOLANGCI_LINT_BIN):
	mkdir -p $(shell pwd)/bin
	hack/golangci-lint.sh $(GOLANGCI_LINT_BIN)
	
.PHONY: build.image
build.image: build.go verify
	$(CONTAINER_ENGINE) build -t ${IMG} .

.PHONY: push.image
push.image: build.image
	$(CONTAINER_ENGINE) tag ${IMG} $(TARGET_REPO)/${IMG}:$(IMAGE_TAG)
	$(CONTAINER_ENGINE) push ${TARGET_REPO}/${IMG}:$(IMAGE_TAG)