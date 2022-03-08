

ifeq (,$(shell which podman 2>/dev/null))
CONTAINER_ENGINE ?= docker
else
CONTAINER_ENGINE ?= podman
endif

# Image URL to use all building/pushing image targets
IMG ?= node-observability-agent:go-latest
TARGET_REPO ?= quay.io/skhoury
GOLANGCI_LINT_VERSION = v1.42.1
GOLANGCI_LINT_BIN=$(BIN_DIR)/golangci-lint
COVERPROFILE = coverage.out
BIN_DIR=$(shell pwd)/bin

.PHONY: vendors
vendors:
	@echo "### Checking vendors"
	go mod tidy && go mod vendor

.PHONY: fmt
fmt:
	go fmt -mod vendor ./...

#.PHONY: lint
#lint:  
## Checks the code with golangci-lint
#lint: $(GOLANGCI_LINT_BIN)
#	$(GOLANGCI_LINT_BIN) run -c .golangci.yaml --deadline=30m

#$(GOLANGCI_LINT_BIN):
#	mkdir -p $(BIN_DIR)
#	hack/golangci-lint.sh $(GOLANGCI_LINT_BIN)

.PHONY: test
test: vendor fmt #lint
	@echo "### Testing"
	go test -mod vendor ./... -timeout 50s -coverprofile ${COVERPROFILE} -tags fake

.PHONY: verify
verify: test #lint test 

.PHONY: build.go
build.go: vendor fmt #lint
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
