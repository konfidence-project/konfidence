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
PNPM ?= pnpm

STAR_SAMPLE_DIR ?= test/data/samples/star
GALAXY_SAMPLE_DIR ?= test/data/samples/galaxy
STAR_CRD_DIR ?= test/data/crds/star
GALAXY_CRD_DIR ?= test/data/crds/galaxy
SCHEMA_DIR ?= $(REPO_ROOT)/.tmp/schemas
# Staging dir for the full CRD set. galaxy and star share one API group
# (konfidence.cloud), so all CRDs are generated once here and then split into
# the per-operator dirs / charts by kind (see STAR_CRD_FILES / GALAXY_CRD_FILES).
CRD_STAGING_DIR ?= $(REPO_ROOT)/.tmp/crds

# Merged API package (single group konfidence.cloud).
API_PATHS = paths="./api/v1alpha1/..."

# Which generated CRD files belong to which operator chart. All are in the
# konfidence.cloud group; galaxy vs. star stays a code/deployment convention.
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

# Internal controller packages per operator. internal/ is flat (no galaxy/star
# grouping dir), so each operator's controllers are enumerated explicitly.
STAR_INTERNAL_DIRS    = ./internal/stage/... ./internal/taskorchestration/... ./internal/vectoractivation/... ./internal/vectordeployment/...
GALAXY_INTERNAL_DIRS  = ./internal/stageconfiguration/... ./internal/vectorassembly/... ./internal/vectorpromotion/...
STAR_INTERNAL_PATHS   = $(foreach d,$(STAR_INTERNAL_DIRS),paths="$(d)")
GALAXY_INTERNAL_PATHS = $(foreach d,$(GALAXY_INTERNAL_DIRS),paths="$(d)")

# Kubernetes / envtest versions
ENVTEST_K8S_VERSION ?= 1.33

DEX_COMPOSE_FILE ?= dev/dex/docker-compose.yaml
DEX_ISSUER       ?= http://localhost:5556/dex
API_AUTH_FLAGS   ?= --auth-authorize-url $(DEX_ISSUER)/auth \
	--auth-token-url $(DEX_ISSUER)/token \
	--auth-userinfo-url $(DEX_ISSUER)/userinfo \
	--auth-client-id kden-local \
	--auth-redirect-uri http://localhost:8090/api/auth/callback

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
STAR_IMAGE   = $(REGISTRY)/star-operator:$(TAG)
GALAXY_IMAGE = $(REGISTRY)/galaxy-operator:$(TAG)
API_IMAGE    = $(REGISTRY)/api:$(TAG)

.PHONY: all
all: api build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: hermit manifests-star manifests-galaxy ## Generate CRDs and RBAC manifests for all operators.

.PHONY: manifests-crds
manifests-crds: hermit ## Generate the full CRD set (single konfidence.cloud group) into the staging dir.
	@echo "Generating CRDs for the konfidence.cloud group..."
	@rm -rf $(CRD_STAGING_DIR)
	@mkdir -p $(CRD_STAGING_DIR)
	$(CONTROLLER_GEN) crd $(API_PATHS) output:crd:artifacts:config=$(CRD_STAGING_DIR)
	@# Guard: every generated CRD must be assigned to exactly one operator chart.
	@# Without this, a newly added CRD kind would be silently dropped from both
	@# charts (galaxy vs. star membership is enumerated, not inferred).
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

.PHONY: manifests-star
manifests-star: hermit manifests-crds ## Generate star RBAC and distribute star CRDs to the star chart.
	@echo "Generating manifests for star..."
	@mkdir -p $(STAR_CRD_DIR) charts/star/templates/crds config/rbac/star
	$(CONTROLLER_GEN) rbac:roleName=star-manager \
		$(STAR_INTERNAL_PATHS) \
		output:rbac:artifacts:config=config/rbac/star
	@rm -f $(STAR_CRD_DIR)/*.yaml charts/star/templates/crds/*.yaml
	@for f in $(STAR_CRD_FILES); do cp "$(CRD_STAGING_DIR)/$$f" "$(STAR_CRD_DIR)/$$f"; done
	for f in $(STAR_CRD_DIR)/*.yaml; do \
		charts/patch-crd.sh star "$$f" "charts/star/templates/crds/$$(basename $$f)"; \
	done
	charts/patch-clusterrole.sh star "config/rbac/star/role.yaml" "charts/star/templates/clusterrole.yaml"
	$(HELM_DOCS) -c charts/star > charts/star/README.md

.PHONY: manifests-galaxy
manifests-galaxy: hermit manifests-crds ## Generate galaxy RBAC and distribute galaxy CRDs to the galaxy chart.
	@echo "Generating manifests for galaxy..."
	@mkdir -p $(GALAXY_CRD_DIR) charts/galaxy/templates/crds config/rbac/galaxy
	$(CONTROLLER_GEN) rbac:roleName=galaxy-manager \
		$(GALAXY_INTERNAL_PATHS) \
		output:rbac:artifacts:config=config/rbac/galaxy
	@rm -f $(GALAXY_CRD_DIR)/*.yaml charts/galaxy/templates/crds/*.yaml
	@for f in $(GALAXY_CRD_FILES); do cp "$(CRD_STAGING_DIR)/$$f" "$(GALAXY_CRD_DIR)/$$f"; done
	for f in $(GALAXY_CRD_DIR)/*.yaml; do \
		charts/patch-crd.sh galaxy "$$f" "charts/galaxy/templates/crds/$$(basename $$f)"; \
	done
	charts/patch-clusterrole.sh galaxy "config/rbac/galaxy/role.yaml" "charts/galaxy/templates/clusterrole.yaml"
	$(HELM_DOCS) -c charts/galaxy > charts/galaxy/README.md

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

.PHONY: lint
lint: hermit ## Run golangci-lint against the entire codebase.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: hermit ## Run golangci-lint and apply automatic fixes.
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: hermit ## Verify the golangci-lint configuration.
	$(GOLANGCI_LINT) config verify

.PHONY: lint-ui
lint-ui: hermit ## Run UI linting.
	$(PNPM) ui:lint

.PHONY: fmt-check-ui
fmt-check-ui: hermit ## Verify UI formatting.
	$(PNPM) ui:fmt:check

.PHONY: verify
verify: fmt-check lint lint-config lint-ui fmt-check-ui test ## Run formatting checks, linting, and all tests.

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
validate: validate-star validate-galaxy ## Validate all sample resources against their JSON schemas.

.PHONY: validate-star
validate-star: schemas ## Validate star sample resources against their JSON schemas.
	@kubeconform -summary \
		-schema-location default \
		-schema-location "$(SCHEMA_DIR)/{{.Group}}_{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
		$(STAR_SAMPLE_DIR)

.PHONY: validate-galaxy
validate-galaxy: schemas ## Validate galaxy sample resources against their JSON schemas.
	@kubeconform -summary \
		-schema-location default \
		-schema-location "$(SCHEMA_DIR)/{{.Group}}_{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
		$(GALAXY_SAMPLE_DIR)

.PHONY: helm-lint
helm-lint: helm-lint-star helm-lint-galaxy ## Run helm lint against all charts.

.PHONY: helm-lint-star
helm-lint-star: hermit ## Run helm lint against the star chart (includes api templates).
	$(HELM) lint charts/star

.PHONY: helm-lint-galaxy
helm-lint-galaxy: hermit ## Run helm lint against the galaxy chart.
	$(HELM) lint charts/galaxy

## Tool Binaries (Testing)
GINKGO ?= $(LOCALBIN)/ginkgo

##@ Testing

.PHONY: test
test: hermit manifests generate fmt vet test-star test-galaxy test-pkg test-kden-cli test-api test-ui ## Run all unit tests.

.PHONY: test-star
test-star: hermit manifests setup-envtest ginkgo ## Run unit tests for the star operator only.
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		$(GINKGO) --coverprofile=cover-star.out -v $(STAR_INTERNAL_DIRS)

.PHONY: test-galaxy
test-galaxy: hermit manifests setup-envtest ginkgo ## Run unit tests for the galaxy operator only.
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		$(GINKGO) --coverprofile=cover-galaxy.out -v $(GALAXY_INTERNAL_DIRS)

.PHONY: test-pkg
test-pkg: hermit ginkgo ## Run unit tests for shared pkg packages.
	$(GINKGO) --coverprofile=cover-pkg.out -v ./pkg/...

.PHONY: test-kden-cli
test-kden-cli: hermit
	go test ./cmd/kden/... ./internal/kden/...

.PHONY: test-api
test-api: hermit fmt vet ginkgo ## Run unit tests for the API server and kden API client.
	$(GINKGO) --coverprofile=cover-api.out -v ./internal/api/... ./internal/kden/apiclient/...

.PHONY: test-ui
test-ui: hermit ## Run UI tests.
	$(PNPM) ui:test

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
build: manifests generate fmt vet build-star build-galaxy build-kden-cli build-api build-ui ## Build all binaries and UI assets.

.PHONY: build-star
build-star: hermit ## Build the star operator binary.
	GORELEASER_CURRENT_TAG=dev goreleaser build --clean --snapshot --single-target --id star -o bin/star

.PHONY: build-galaxy
build-galaxy: hermit ## Build the galaxy operator binary.
	GORELEASER_CURRENT_TAG=dev goreleaser build --clean --snapshot --single-target --id galaxy -o bin/galaxy

.PHONY: build-kden-cli
build-kden-cli: hermit ## Build the kden cli binary.
	GORELEASER_CURRENT_TAG=dev goreleaser build --clean --snapshot --single-target --id kden -o bin/kden

.PHONY: build-api
build-api: hermit ## Build the Konfidence API server binary.
	go build -o bin/api ./cmd/api/main.go

.PHONY: build-ui
build-ui: hermit ## Build the UI app.
	$(PNPM) ui:build

.PHONY: dev-ui
dev-ui: hermit ## Run the UI development server.
	$(PNPM) ui:dev

.PHONY: run-star
run-star: manifests-star generate fmt vet ## Run the star operator from your host.
	go run ./cmd/star/main.go

.PHONY: run-galaxy
run-galaxy: manifests-galaxy generate fmt vet ## Run the galaxy operator from your host.
	go run ./cmd/galaxy/main.go

.PHONY: run-api
run-api: hermit ## Run the API server locally. Set KUBECONFIG for domain endpoints, not needed for probes.
	go run ./cmd/api/main.go

.PHONY: run-api-with-idp
run-api-with-idp: hermit ## Run the API server locally with the local Dex IDP.
	go run ./cmd/api/main.go $(API_AUTH_FLAGS)

.PHONY: idp-up
idp-up: ## Start the local Dex IDP for API authentication development.
	$(CONTAINER_TOOL) compose -f $(DEX_COMPOSE_FILE) up -d

.PHONY: idp-down
idp-down: ## Stop the local Dex IDP.
	$(CONTAINER_TOOL) compose -f $(DEX_COMPOSE_FILE) down

.PHONY: idp-logs
idp-logs: ## Follow logs from the local Dex IDP.
	$(CONTAINER_TOOL) compose -f $(DEX_COMPOSE_FILE) logs -f dex

# These targets are only used for local environments (not in pipeline)
.PHONY: docker-build
docker-build: docker-build-star docker-build-galaxy docker-build-api ## Build container images for all operators (local use only).

.PHONY: docker-build-star
docker-build-star: hermit ## Build the star operator container image (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile --build-arg OPERATOR_NAME=star -t $(STAR_IMAGE) .

.PHONY: docker-build-galaxy
docker-build-galaxy: hermit ## Build the galaxy operator container image (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile --build-arg OPERATOR_NAME=galaxy -t $(GALAXY_IMAGE) .

.PHONY: docker-build-api
docker-build-api: hermit ## Build the Konfidence API server container image (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile --build-arg OPERATOR_NAME=api -t $(API_IMAGE) .

.PHONY: docker-bake
docker-bake: hermit ## Build all container images using docker buildx bake (multi-platform, CI-compatible).
	$(CONTAINER_TOOL) buildx bake --file docker-bake.hcl

.PHONY: docker-push
docker-push: docker-push-star docker-push-galaxy docker-push-api ## Push container images for all operators.

.PHONY: docker-push-star
docker-push-star: ## Push the star operator container image.
	$(CONTAINER_TOOL) push $(STAR_IMAGE)

.PHONY: docker-push-galaxy
docker-push-galaxy: ## Push the galaxy operator container image.
	$(CONTAINER_TOOL) push $(GALAXY_IMAGE)

.PHONY: docker-push-api
docker-push-api: ## Push the Konfidence API server container image.
	$(CONTAINER_TOOL) push $(API_IMAGE)

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: install-star install-galaxy ## Install CRDs for all operators into the cluster specified in ~/.kube/config.

.PHONY: install-star
install-star: hermit manifests-star ## Install star CRDs into the cluster specified in ~/.kube/config.
	$(HELM) upgrade --install star charts/star --set controller.install=false --set crd.keep=false

.PHONY: install-galaxy
install-galaxy: hermit manifests-galaxy ## Install galaxy CRDs into the cluster specified in ~/.kube/config.
	$(HELM) upgrade --install galaxy charts/galaxy --set controller.install=false --set crd.keep=false

.PHONY: uninstall
uninstall: uninstall-star uninstall-galaxy ## Uninstall CRDs for all operators. Use ignore-not-found=true to suppress errors.

.PHONY: uninstall-star
uninstall-star: hermit ## Uninstall star CRDs from the cluster. Use ignore-not-found=true to suppress errors.
	$(HELM) uninstall star --ignore-not-found

.PHONY: uninstall-galaxy
uninstall-galaxy: hermit ## Uninstall galaxy CRDs from the cluster. Use ignore-not-found=true to suppress errors.
	$(HELM) uninstall galaxy --ignore-not-found

.PHONY: deploy
deploy: deploy-star deploy-galaxy ## Deploy all operators to the cluster specified in ~/.kube/config.

.PHONY: deploy-star
deploy-star: hermit manifests-star ## Deploy the star operator to the cluster specified in ~/.kube/config.
	$(HELM) upgrade --install star charts/star \
		--set image.repository=$(REGISTRY)/star-operator \
		--set image.tag=$(TAG) \
		--set crd.keep=false

.PHONY: deploy-galaxy
deploy-galaxy: hermit manifests-galaxy ## Deploy the galaxy operator to the cluster specified in ~/.kube/config.
	$(HELM) upgrade --install galaxy charts/galaxy \
		--set image.repository=$(REGISTRY)/galaxy-operator \
		--set image.tag=$(TAG) \
		--set crd.keep=false

.PHONY: undeploy
undeploy: undeploy-star undeploy-galaxy ## Undeploy all operators. Use ignore-not-found=true to suppress errors.

.PHONY: undeploy-star
undeploy-star: hermit ## Undeploy the star operator. Use ignore-not-found=true to suppress errors.
	$(HELM) uninstall star --ignore-not-found

.PHONY: undeploy-galaxy
undeploy-galaxy: hermit ## Undeploy the galaxy operator. Use ignore-not-found=true to suppress errors.
	$(HELM) uninstall galaxy --ignore-not-found

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
