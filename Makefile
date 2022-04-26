GO=GO111MODULE=on GOFLAGS=-mod=vendor go
GO_BUILD_BINDIR := bin
GO_TEST_FLAGS :=-timeout=50s -tags=fake
GO_PACKAGES=./pkg/... ./cmd/...

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

GOLANGCI_LINT_BIN ?= go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.2

.PHONY: verify
verify:
	hack/verify-deps.sh
	hack/verify-generated.sh
	hack/verify-gofmt.sh
	hack/verify-bundle.sh

.PHONY: lint
## Checks the code with golangci-lint
lint:
	$(GOLANGCI_LINT_BIN) run -c .golangci.yaml --deadline=30m

.PHONY: build.image
build.image: test verify lint
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
	oc project node-observability-operator || oc new-project node-observability-operator
	IMG=$(IMG) hack/kustomize-build.sh | oc apply -f -
