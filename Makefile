# Image registry and tag used by all build/push targets
REGISTRY ?= registry.kdenv.lab
TAG      ?= dev
DASHBOARD_ENABLED ?= false

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
PNPM ?= pnpm

STAR_SAMPLE_DIR ?= test/data/samples/star
GALAXY_SAMPLE_DIR ?= test/data/samples/galaxy
STAR_CRD_DIR ?= test/data/crds/star
GALAXY_CRD_DIR ?= test/data/crds/galaxy
SCHEMA_DIR ?= $(REPO_ROOT)/.tmp/schemas
# Staging dir for the full CRD set. galaxy and star share one API group
# (konfidence.cloud), so all CRDs are generated once here and then split into
# the per-set dirs / chart templates by kind (see STAR_CRD_FILES / GALAXY_CRD_FILES).
CRD_STAGING_DIR ?= $(REPO_ROOT)/.tmp/crds

# Merged API package (single group konfidence.cloud).
API_PATHS = paths="./api/v1alpha1/..."

# Which generated CRD files belong to the galaxy vs. star controller set.
# All are in the konfidence.cloud group and ship in the one konfidence
# chart; the split decides which install guard each CRD template gets
# (galaxy CRDs are additionally gated on .Values.galaxy.enabled) and
# which test/data/crds dir it lands in (the envtest suites load them
# per set).
GALAXY_CRD_FILES = \
	konfidence.cloud_stageconfigurations.yaml \
	konfidence.cloud_vectorpromotions.yaml \
	konfidence.cloud_vectorpromotionconfigs.yaml \
	konfidence.cloud_vectortemplates.yaml
STAR_CRD_FILES = \
	konfidence.cloud_stages.yaml \
	konfidence.cloud_stageversions.yaml \
	konfidence.cloud_stageversionusages.yaml \
	konfidence.cloud_artifactdeployments.yaml \
	konfidence.cloud_vectoractivations.yaml \
	konfidence.cloud_vectorassignments.yaml \
	konfidence.cloud_vectordata.yaml \
	konfidence.cloud_vectordeployments.yaml \
	konfidence.cloud_vectormigrations.yaml \
	konfidence.cloud_taskexecutions.yaml \
	konfidence.cloud_activationtaskexecutions.yaml \
	konfidence.cloud_activationtaskregistrations.yaml

# Internal controller packages of the konfidence operator. internal/ is flat
# (no galaxy/star grouping dir), so the controllers are enumerated explicitly.
# RBAC is generated per set so the chart can gate the galaxy-only rules on
# .Values.galaxy.enabled; tests run over the combined list.
STAR_INTERNAL_DIRS    = ./internal/stage/... ./internal/taskorchestration/... ./internal/vectoractivation/... ./internal/vectordeployment/...
GALAXY_INTERNAL_DIRS  = ./internal/stageconfiguration/... ./internal/vectorassembly/... ./internal/vectorpromotion/...
OPERATOR_INTERNAL_DIRS = $(STAR_INTERNAL_DIRS) $(GALAXY_INTERNAL_DIRS)
STAR_INTERNAL_PATHS   = $(foreach d,$(STAR_INTERNAL_DIRS),paths="$(d)")
GALAXY_INTERNAL_PATHS = $(foreach d,$(GALAXY_INTERNAL_DIRS),paths="$(d)")

# Kubernetes / envtest versions
ENVTEST_K8S_VERSION ?= 1.33

DEX_CONTAINER    ?= konfidence-dex
DEX_ISSUER       ?= http://localhost:5556/dex
API_AUTH_FLAGS   ?= --auth-authorize-url $(DEX_ISSUER)/auth \
	--auth-token-url $(DEX_ISSUER)/token \
	--auth-userinfo-url $(DEX_ISSUER)/userinfo \
	--auth-client-id kden-local \
	--auth-redirect-uri http://localhost:5173/api/auth/callback

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
UI_IMAGE       = $(REGISTRY)/konfidence-ui:$(TAG)

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
	@mkdir -p $(STAR_CRD_DIR) $(GALAXY_CRD_DIR) charts/konfidence/templates/crds config/rbac/star config/rbac/galaxy
	$(CONTROLLER_GEN) rbac:roleName=konfidence-manager \
		$(STAR_INTERNAL_PATHS) \
		output:rbac:artifacts:config=config/rbac/star
	$(CONTROLLER_GEN) rbac:roleName=konfidence-manager \
		$(GALAXY_INTERNAL_PATHS) \
		output:rbac:artifacts:config=config/rbac/galaxy
	@rm -f $(STAR_CRD_DIR)/*.yaml $(GALAXY_CRD_DIR)/*.yaml charts/konfidence/templates/crds/*.yaml
	@for f in $(STAR_CRD_FILES); do cp "$(CRD_STAGING_DIR)/$$f" "$(STAR_CRD_DIR)/$$f"; done
	@for f in $(GALAXY_CRD_FILES); do cp "$(CRD_STAGING_DIR)/$$f" "$(GALAXY_CRD_DIR)/$$f"; done
	for f in $(STAR_CRD_DIR)/*.yaml; do \
		charts/patch-crd.sh konfidence "$$f" "charts/konfidence/templates/crds/$$(basename $$f)"; \
	done
	for f in $(GALAXY_CRD_DIR)/*.yaml; do \
		charts/patch-crd.sh konfidence "$$f" "charts/konfidence/templates/crds/$$(basename $$f)" \
			"and .Values.crd.install .Values.galaxy.enabled"; \
	done
	charts/patch-clusterrole.sh konfidence "config/rbac/star/role.yaml" "charts/konfidence/templates/clusterrole.yaml" \
		"config/rbac/galaxy/role.yaml" ".Values.galaxy.enabled"
	$(HELM_DOCS) -c charts/konfidence > charts/konfidence/README.md

.PHONY: manifests-crds
manifests-crds: hermit ## Generate the full CRD set (single konfidence.cloud group) into the staging dir.
	@echo "Generating CRDs for the konfidence.cloud group..."
	@rm -rf $(CRD_STAGING_DIR)
	@mkdir -p $(CRD_STAGING_DIR)
	$(CONTROLLER_GEN) crd $(API_PATHS) output:crd:artifacts:config=$(CRD_STAGING_DIR)
	@# Guard: every generated CRD must be assigned to exactly one controller set.
	@# Without this, a newly added CRD kind would be silently dropped from the
	@# chart (galaxy vs. star membership is enumerated, not inferred).
	@assigned=" $(STAR_CRD_FILES) $(GALAXY_CRD_FILES) "; unassigned=""; \
	for f in $(CRD_STAGING_DIR)/*.yaml; do \
		b=$$(basename "$$f"); \
		case "$$assigned" in *" $$b "*) ;; *) unassigned="$$unassigned $$b";; esac; \
	done; \
	if [ -n "$$unassigned" ]; then \
		echo "ERROR: generated CRD(s) not assigned to STAR_CRD_FILES or GALAXY_CRD_FILES:$$unassigned" >&2; \
		echo "Add each to the matching list in the Makefile." >&2; \
		exit 1; \
	fi

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

.PHONY: fmt-go
fmt-go: hermit
	go fmt ./...

.PHONY: fmt
fmt: fmt-go ## Run Go and UI formatters.
	$(PNPM) ui:fmt

.PHONY: fmt-check
fmt-check: hermit ## Verify go formatting across the entire codebase.
	@test -z "$$(gofmt -l .)" || { \
		echo "Go files need formatting:"; \
		gofmt -l .; \
		exit 1; \
	}

.PHONY: vet
vet: hermit ## Run go vet against the entire codebase.
	go vet ./...

.PHONY: lint-go
lint-go: hermit
	$(GOLANGCI_LINT) run

.PHONY: lint
lint: lint-go ## Run Go and UI linters.
	$(PNPM) ui:lint

.PHONY: lint-fix-go
lint-fix-go: hermit
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-fix
lint-fix: lint-fix-go ## Run Go and UI linters and apply automatic fixes.
	$(PNPM) ui:lint:fix

.PHONY: lint-config
lint-config: hermit ## Verify the golangci-lint configuration.
	$(GOLANGCI_LINT) config verify

.PHONY: verify-ui
verify-ui: ## Run UI typecheck, lint, Svelte checks, and format check.
	$(PNPM) ui:verify

.PHONY: verify
verify: fmt-check lint-go lint-config verify-ui test-operators test-pkg ## Run Go checks/tests and UI verification.

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
	@for crd in $(STAR_CRD_DIR)/*.yaml $(GALAXY_CRD_DIR)/*.yaml; do \
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
		$(STAR_SAMPLE_DIR) $(GALAXY_SAMPLE_DIR)

.PHONY: helm-lint
helm-lint: hermit ## Run helm lint against the konfidence chart.
	$(HELM) lint charts/konfidence

## Tool Binaries (Testing)
GINKGO ?= $(LOCALBIN)/ginkgo

##@ Testing

.PHONY: test
test: hermit manifests generate fmt-go vet test-operators test-pkg test-kden-cli test-api ## Run all Go unit tests.

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
test-api: hermit fmt-go vet ginkgo ## Run unit tests for the API server and kden API client.
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
build: manifests generate fmt vet build-operator build-kden-cli ## Build all binaries and UI assets.
	$(PNPM) ui:build

.PHONY: build-operator
build-operator: hermit ## Build the konfidence operator binary.
	GORELEASER_CURRENT_TAG=dev goreleaser build --clean --snapshot --single-target --id konfidence -o bin/konfidence

.PHONY: build-kden-cli
build-kden-cli: hermit ## Build the kden cli binary.
	GORELEASER_CURRENT_TAG=dev goreleaser build --clean --snapshot --single-target --id kden -o bin/kden

.PHONY: run
run: manifests generate fmt vet ## Run the konfidence operator from your host.
	go run ./cmd/konfidence/main.go

.PHONY: build-api
build-api: hermit ## Build the Konfidence API server binary.
	go build -o bin/api ./cmd/api/main.go

.PHONY: run-api
run-api: hermit ## Run the API server locally. Set KUBECONFIG for domain endpoints, not needed for probes.
	go run ./cmd/api/main.go

.PHONY: run-api-with-idp
run-api-with-idp: hermit ## Run the API server locally against the development Dex provider.
	go run ./cmd/api/main.go $(API_AUTH_FLAGS)

.PHONY: idp-up
idp-up: ## Start the development Dex provider.
	@$(CONTAINER_TOOL) rm --force $(DEX_CONTAINER) >/dev/null 2>&1 || true
	$(CONTAINER_TOOL) run --detach --name $(DEX_CONTAINER) \
		--publish 5556:5556 \
		--volume $(REPO_ROOT)/dev/dex/config.yaml:/etc/dex/config.yaml:ro \
		ghcr.io/dexidp/dex:v2.45.1 dex serve /etc/dex/config.yaml

.PHONY: idp-down
idp-down: ## Stop the development Dex provider.
	@$(CONTAINER_TOOL) rm --force $(DEX_CONTAINER) >/dev/null 2>&1 || true

.PHONY: idp-logs
idp-logs: ## Follow logs from the development Dex provider.
	$(CONTAINER_TOOL) logs --follow $(DEX_CONTAINER)

# These targets are only used for local environments (not in pipeline)
.PHONY: docker-build
docker-build: hermit docker-build-ui ## Build all container images (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile --build-arg OPERATOR_NAME=konfidence -t $(OPERATOR_IMAGE) .

.PHONY: docker-build-ui
docker-build-ui: hermit ## Build the dashboard container image (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile.ui -t $(UI_IMAGE) .

.PHONY: docker-bake
docker-bake: hermit ## Build all container images using docker buildx bake (multi-platform, CI-compatible).
	$(CONTAINER_TOOL) buildx bake --file docker-bake.hcl

.PHONY: docker-push
docker-push: ## Push all container images.
	$(CONTAINER_TOOL) push $(OPERATOR_IMAGE)
	$(CONTAINER_TOOL) push $(UI_IMAGE)

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
		--set dashboard.image.repository=$(REGISTRY)/konfidence-ui \
		--set image.tag=$(TAG) \
		--set dashboard.image.tag=$(TAG) \
		--set dashboard.enabled=$(DASHBOARD_ENABLED) \
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
