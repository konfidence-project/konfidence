# Image registry and tag used by all build/push targets
REGISTRY ?= registry.kdenv.lab
TAG      ?= dev

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool used for building images.
# Replace with podman or another tool if needed.
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Add Hermit bin directory to PATH for all make targets
export PATH := $(shell pwd)/bin:$(PATH)

REPO_ROOT := $(shell git rev-parse --show-toplevel)

SAMPLE_DIR ?= test/data/samples
CRD_DIR ?= test/data/crds
SCHEMA_DIR ?= $(REPO_ROOT)/.tmp/schemas
# Staging dir for the full CRD set (konfidence.cloud group).
CRD_STAGING_DIR ?= $(REPO_ROOT)/.tmp/crds

# Merged API package (single group konfidence.cloud).
API_PATHS = paths="./api/v1alpha1/..."

# Internal controller packages of the konfidence operator.
# Auto-discover by finding all internal/ subdirs containing setup.go, then append /internal/controller
OPERATOR_INTERNAL_DIRS = $(shell find internal -maxdepth 2 -name setup.go -exec dirname {} \; | sed 's|^|./|; s|$$|/internal/controller|' | sort)
OPERATOR_INTERNAL_PATHS = $(foreach d,$(OPERATOR_INTERNAL_DIRS),paths="$(d)")

# Kubernetes / envtest versions
ENVTEST_K8S_VERSION ?= 1.33

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL        ?= kubectl
KIND           ?= kind
CONTROLLER_GEN ?= controller-gen
ENVTEST        ?= setup-envtest
GOLANGCI_LINT   = golangci-lint
HELM           ?= helm
HELM_DOCS      ?= helm-docs

## Image names
OPERATOR_IMAGE = $(REGISTRY)/konfidence-operator:$(TAG)

.PHONY: all
all: api build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: hermit manifests-crds ## Generate CRDs and RBAC manifests for the konfidence operator chart.
	@echo "Generating manifests for konfidence..."
	@mkdir -p $(CRD_DIR) charts/konfidence/templates/crds config/rbac
	$(CONTROLLER_GEN) rbac:roleName=konfidence-manager \
		$(OPERATOR_INTERNAL_PATHS) \
		output:rbac:artifacts:config=config/rbac
	@rm -f $(CRD_DIR)/*.yaml charts/konfidence/templates/crds/*.yaml
	@cp $(CRD_STAGING_DIR)/*.yaml $(CRD_DIR)/
	for f in $(CRD_DIR)/*.yaml; do \
		charts/patch-crd.sh konfidence "$$f" "charts/konfidence/templates/crds/$$(basename $$f)"; \
	done
	charts/patch-clusterrole.sh konfidence config/rbac/role.yaml charts/konfidence/templates/clusterrole.yaml
	$(HELM_DOCS) -c charts/konfidence > charts/konfidence/README.md

.PHONY: manifests-crds
manifests-crds: hermit ## Generate the full CRD set (single konfidence.cloud group) into the staging dir.
	@echo "Generating CRDs for the konfidence.cloud group..."
	@rm -rf $(CRD_STAGING_DIR)
	@mkdir -p $(CRD_STAGING_DIR)
	$(CONTROLLER_GEN) crd $(API_PATHS) output:crd:artifacts:config=$(CRD_STAGING_DIR)

.PHONY: generate
generate: hermit ## Generate DeepCopy implementations for the merged API package.
	$(CONTROLLER_GEN) object $(API_PATHS)

.PHONY: generate-mocks
generate-mocks: hermit ## Regenerate all gomock mocks via go generate.
	go generate ./...

.PHONY: regenerate-mocks
regenerate-mocks: hermit ## Wipe every mocks/ directory's contents, then regenerate from go:generate directives.
	find . -path ./vendor -prune -o -type d -name mocks -print -exec sh -c 'rm -rf "$$0"/*' {} \;
	$(MAKE) generate-mocks

.PHONY: fmt
fmt: hermit ## Run go fmt against the entire codebase.
	go fmt ./...

.PHONY: vet
vet: hermit ## Run go vet against the entire codebase.
	go vet ./...

.PHONY: lint
lint: hermit ## Run golangci-lint against the entire codebase.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: hermit ## Run golangci-lint and apply automatic fixes.
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: hermit ## Verify the golangci-lint configuration.
	$(GOLANGCI_LINT) config verify

##@ API

.PHONY: api
api: hermit manifests generate docs schemas helm-lint ## Run full API generation pipeline (manifests, deepcopy, docs, schemas, helm lint).

.PHONY: docs
docs: hermit ## Generate CRD reference documentation for the konfidence.cloud API.
	@echo "Generating CRD documentation..."
	@mkdir -p api/docs
	@crd-ref-docs \
		--source-path="api/v1alpha1" \
		--config="$(REPO_ROOT)/api/.crd-ref-docs.config.yaml" \
		--output-path="api/docs" \
		--output-mode=group \
		--renderer=markdown
	@if [ -f "api/docs/konfidence.cloud.md" ]; then \
		mv "api/docs/konfidence.cloud.md" "api/docs/README.md"; \
	fi

.PHONY: schemas
schemas: hermit manifests ## Extract JSON schemas for each CRD version.
	@rm -rf $(SCHEMA_DIR)
	@mkdir -p $(SCHEMA_DIR)
	@for crd in $(CRD_DIR)/*.yaml; do \
		crd_kind=$$(yq ".spec.names.kind" $$crd | tr '[:upper:]' '[:lower:]'); \
		crd_group="$$(yq ".spec.group" $$crd)"; \
		for ver in $$(yq -r '.spec.versions[].name' $$crd); do \
			yq -o=json ".spec.versions[] | select(.name == \"$$ver\") | .schema.openAPIV3Schema" $$crd \
				> "$(SCHEMA_DIR)/$${crd_group}_$${crd_kind}_$${ver}.json"; \
		done; \
	done

.PHONY: validate
validate: schemas ## Validate all sample resources against their JSON schemas.
	@kubeconform -summary \
		-schema-location default \
		-schema-location "$(SCHEMA_DIR)/{{.Group}}_{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
		$(SAMPLE_DIR)

.PHONY: helm-lint
helm-lint: hermit ## Run helm lint against the konfidence chart.
	$(HELM) lint charts/konfidence

## Tool Binaries (Testing)
GINKGO ?= $(LOCALBIN)/ginkgo

##@ Testing

.PHONY: test
test: hermit manifests generate fmt vet test-operators test-pkg test-kden-cli test-api ## Run all unit tests.

.PHONY: test-operators
test-operators: hermit manifests setup-envtest ginkgo ## Run unit tests for the konfidence operator.
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		$(GINKGO) --coverprofile=cover-operators.out -v $(OPERATOR_INTERNAL_DIRS) ./cmd/konfidence/...

.PHONY: test-pkg
test-pkg: hermit ginkgo ## Run unit tests for shared pkg packages.
	$(GINKGO) --coverprofile=cover-pkg.out -v ./pkg/...

.PHONY: test-kden-cli
test-kden-cli: hermit
	go test ./cmd/kden/... ./internal/kden/...

.PHONY: test-api
test-api: hermit fmt vet ginkgo ## Run unit tests for the API server and kden API client.
	$(GINKGO) --coverprofile=cover-api.out -v ./internal/api/... ./internal/kden/apiclient/...

.PHONY: setup-envtest
setup-envtest: hermit ## Download the envtest binaries for the configured Kubernetes version.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: ginkgo
ginkgo: ## Install ginkgo CLI to LOCALBIN.
	go build -o $(LOCALBIN)/ginkgo github.com/onsi/ginkgo/v2/ginkgo

##@ Build

.PHONY: build
build: manifests generate fmt vet build-operator build-kden-cli ## Build all binaries.

.PHONY: build-operator
build-operator: hermit ## Build the konfidence operator binary.
	GORELEASER_CURRENT_TAG=dev goreleaser build --clean --snapshot --single-target --id konfidence -o bin/konfidence

.PHONY: build-kden-cli
build-kden-cli: hermit ## Build the kden cli binary.
	GORELEASER_CURRENT_TAG=dev goreleaser build --clean --snapshot --single-target --id kden -o bin/kden

.PHONY: run
run: manifests generate fmt vet ## Run the konfidence operator from your host.
	go run ./cmd/konfidence/main.go

.PHONY: run-kden-api
run-kden-api: fmt vet ## Run the kden API server locally.
	go run ./cmd/api/main.go

# These targets are only used for local environments (not in pipeline)
.PHONY: docker-build
docker-build: hermit ## Build the konfidence operator container image (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile --build-arg OPERATOR_NAME=konfidence -t $(OPERATOR_IMAGE) .

.PHONY: docker-bake
docker-bake: hermit ## Build all container images using docker buildx bake (multi-platform, CI-compatible).
	$(CONTAINER_TOOL) buildx bake --file docker-bake.hcl

.PHONY: docker-push
docker-push: ## Push the konfidence operator container image.
	$(CONTAINER_TOOL) push $(OPERATOR_IMAGE)

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: hermit manifests ## Install CRDs into the cluster specified in ~/.kube/config.
	$(HELM) upgrade --install konfidence charts/konfidence --set controller.install=false --set crd.keep=false

.PHONY: uninstall
uninstall: hermit ## Uninstall CRDs from the cluster. Use ignore-not-found=true to suppress errors.
	$(HELM) uninstall konfidence --ignore-not-found

.PHONY: deploy
deploy: hermit manifests ## Deploy the konfidence operator to the cluster specified in ~/.kube/config.
	$(HELM) upgrade --install konfidence charts/konfidence \
		--set image.repository=$(REGISTRY)/konfidence-operator \
		--set image.tag=$(TAG) \
		--set crd.keep=false

.PHONY: undeploy
undeploy: hermit ## Undeploy the konfidence operator. Use ignore-not-found=true to suppress errors.
	$(HELM) uninstall konfidence --ignore-not-found

##@ Developer Setup

.PHONY: hermit
hermit: ## Check that Hermit is installed and its environment is activated.
	@command -v hermit >/dev/null 2>&1 || { \
		echo "Hermit is not installed. Please install it from https://cashapp.github.io/hermit/"; \
		exit 1; \
	}
	@hermit status >/dev/null 2>&1 || { \
		echo "Hermit environment is not activated. Run 'source ./bin/activate-hermit' or 'eval \"\$$(hermit env)\"'"; \
		exit 1; \
	}

.PHONY: install-git-hooks
install-git-hooks: hermit ## Install git hooks via prek.
	@echo "Setting up prek (pre-commit) installing git hooks..."
	prek install

.PHONY: uninstall-git-hooks
uninstall-git-hooks: hermit ## Uninstall git hooks via prek.
	@echo "Uninstalling prek (pre-commit) git hooks..."
	prek uninstall
