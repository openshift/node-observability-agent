

ifeq (,$(shell which podman 2>/dev/null))
CONTAINER_ENGINE ?= docker
else
CONTAINER_ENGINE ?= podman
endif

# Image URL to use all building/pushing image targets
IMG ?= node-observability-agent:go-latest
TARGET_REPO ?= quay.io/skhoury
GOLANGCI_LINT_VERSION = v1.42.1
COVERPROFILE = coverage.out

.PHONY: prereqs
prereqs:
	@echo "### Test if prerequisites are met, and installing missing dependencies"
	test -f $(go env GOPATH)/bin/golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}

.PHONY: vendors
vendors:
	@echo "### Checking vendors"
	go mod tidy && go mod vendor

.PHONY: fmt
fmt:
	go fmt -mod vendor ./...

.PHONY: lint
lint: 
	@echo "### Linting code"
	golangci-lint run ./...

.PHONY: test
test: vendor fmt lint
	@echo "### Testing"
	go test -mod vendor ./... -coverprofile ${COVERPROFILE}

.PHONY: verify
verify: lint test

.PHONY: build.go
build.go: vendor fmt lint
	@echo "### Building"
	go build -mod vendor -o node-observability-agent cmd/node-observability-agent.go

.PHONY: noa
noa: build.image push.image

.PHONY: build.image
build.image: build.go verify
	$(CONTAINER_ENGINE) build -t ${IMG} .

.PHONY: push.image
push.image: build.image
	$(CONTAINER_ENGINE) tag ${IMG} quay.io/skhoury/${IMG}
	$(CONTAINER_ENGINE) push ${TARGET_REPO}/${IMG}
