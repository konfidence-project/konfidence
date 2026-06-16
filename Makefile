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

STAR_SAMPLE_DIR ?= test/data/samples/star
GALAXY_SAMPLE_DIR ?= test/data/samples/galaxy
STAR_CRD_DIR ?= test/data/crds/star
GALAXY_CRD_DIR ?= test/data/crds/galaxy
STAR_SCHEMA_DIR ?= $(REPO_ROOT)/.tmp/schemas/star
GALAXY_SCHEMA_DIR ?= $(REPO_ROOT)/.tmp/schemas/galaxy

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
STAR_IMAGE   = $(REGISTRY)/star-operator:$(TAG)
GALAXY_IMAGE = $(REGISTRY)/galaxy-operator:$(TAG)

.PHONY: all
all: api build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: hermit manifests-star manifests-galaxy ## Generate CRDs and RBAC manifests for all operators.

.PHONY: manifests-star
manifests-star: hermit ## Generate CRDs and RBAC manifests for the star operator.
	@echo "Generating manifests for star..."
	@mkdir -p $(STAR_CRD_DIR) charts/star/templates/crds config/rbac/star
	$(CONTROLLER_GEN) rbac:roleName=star-manager crd webhook \
		paths="./internal/star/..." paths="./api/star/..." \
		output:crd:artifacts:config=$(STAR_CRD_DIR) \
		output:rbac:artifacts:config=config/rbac/star
	for f in $(STAR_CRD_DIR)/*.yaml; do \
		charts/patch-crd.sh star "$$f" "charts/star/templates/crds/$$(basename $$f)"; \
	done
	charts/patch-clusterrole.sh star "config/rbac/star/role.yaml" "charts/star/templates/clusterrole.yaml"
	$(HELM_DOCS) -c charts/star > charts/star/README.md

.PHONY: manifests-galaxy
manifests-galaxy: hermit ## Generate CRDs and RBAC manifests for the galaxy operator.
	@echo "Generating manifests for galaxy..."
	@mkdir -p $(GALAXY_CRD_DIR) charts/galaxy/templates/crds config/rbac/galaxy
	$(CONTROLLER_GEN) rbac:roleName=galaxy-manager crd webhook \
		paths="./internal/galaxy/..." paths="./api/galaxy/..." \
		output:crd:artifacts:config=$(GALAXY_CRD_DIR) \
		output:rbac:artifacts:config=config/rbac/galaxy
	for f in $(GALAXY_CRD_DIR)/*.yaml; do \
		charts/patch-crd.sh galaxy "$$f" "charts/galaxy/templates/crds/$$(basename $$f)"; \
	done
	charts/patch-clusterrole.sh galaxy "config/rbac/galaxy/role.yaml" "charts/galaxy/templates/clusterrole.yaml"
	$(HELM_DOCS) -c charts/galaxy > charts/galaxy/README.md

.PHONY: generate
generate: hermit generate-star generate-galaxy ## Generate DeepCopy implementations for all operators.

.PHONY: generate-star
generate-star: hermit ## Generate DeepCopy implementations for the star operator.
	$(CONTROLLER_GEN) object \
		paths="./internal/star/..." paths="./api/star/..."

.PHONY: generate-galaxy
generate-galaxy: hermit ## Generate DeepCopy implementations for the galaxy operator.
	$(CONTROLLER_GEN) object \
		paths="./internal/galaxy/..." paths="./api/galaxy/..."

.PHONY: generate-mocks
generate-mocks: hermit ## Regenerate all gomock mocks via go generate.
	go generate ./...

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
api: hermit api-star api-galaxy ## Run full API generation pipeline for all operators (manifests, generate, docs, schemas).

.PHONY: api-star
api-star: hermit manifests-star generate-star docs-star schemas-star helm-lint-star ## Run full API generation pipeline for the star operator.

.PHONY: api-galaxy
api-galaxy: hermit manifests-galaxy generate-galaxy docs-galaxy schemas-galaxy helm-lint-galaxy ## Run full API generation pipeline for the galaxy operator.

.PHONY: docs-star
docs-star: hermit ## Generate CRD reference documentation for the star API.
	@echo "Generating CRD documentation for star..."
	@mkdir -p api/star/docs
	@crd-ref-docs \
		--source-path="api/star" \
		--config="$(REPO_ROOT)/api/.crd-ref-docs.config.yaml" \
		--output-path="api/star/docs" \
		--output-mode=group \
		--renderer=markdown
	@if [ -f "api/star/docs/star.konfidence.cloud.md" ]; then \
		mv "api/star/docs/star.konfidence.cloud.md" "api/star/docs/README.md"; \
	fi

.PHONY: docs-galaxy
docs-galaxy: hermit ## Generate CRD reference documentation for the galaxy API.
	@echo "Generating CRD documentation for galaxy..."
	@mkdir -p api/galaxy/docs
	@crd-ref-docs \
		--source-path="api/galaxy" \
		--config="$(REPO_ROOT)/api/.crd-ref-docs.config.yaml" \
		--output-path="api/galaxy/docs" \
		--output-mode=group \
		--renderer=markdown
	@if [ -f "api/galaxy/docs/galaxy.konfidence.cloud.md" ]; then \
		mv "api/galaxy/docs/galaxy.konfidence.cloud.md" "api/galaxy/docs/README.md"; \
	fi

.PHONY: schemas-star
schemas-star: hermit manifests-star ## Extract JSON schemas for each star CRD version.
	@rm -rf $(STAR_SCHEMA_DIR)
	@mkdir -p $(STAR_SCHEMA_DIR)
	@for crd in $(STAR_CRD_DIR)/*.yaml; do \
		crd_kind=$$(yq ".spec.names.kind" $$crd | tr '[:upper:]' '[:lower:]'); \
		crd_group="$$(yq ".spec.group" $$crd)"; \
		for ver in $$(yq -r '.spec.versions[].name' $$crd); do \
			yq -o=json ".spec.versions[] | select(.name == \"$$ver\") | .schema.openAPIV3Schema" $$crd \
				> "$(STAR_SCHEMA_DIR)/$${crd_group}_$${crd_kind}_$${ver}.json"; \
		done; \
	done

.PHONY: schemas-galaxy
schemas-galaxy: hermit manifests-galaxy ## Extract JSON schemas for each galaxy CRD version.
	@rm -rf $(GALAXY_SCHEMA_DIR)
	@mkdir -p $(GALAXY_SCHEMA_DIR)
	@for crd in $(GALAXY_CRD_DIR)/*.yaml; do \
		crd_kind=$$(yq ".spec.names.kind" $$crd | tr '[:upper:]' '[:lower:]'); \
		crd_group="$$(yq ".spec.group" $$crd)"; \
		for ver in $$(yq -r '.spec.versions[].name' $$crd); do \
			yq -o=json ".spec.versions[] | select(.name == \"$$ver\") | .schema.openAPIV3Schema" $$crd \
				> "$(GALAXY_SCHEMA_DIR)/$${crd_group}_$${crd_kind}_$${ver}.json"; \
		done; \
	done

.PHONY: validate
validate: validate-star validate-galaxy ## Validate all sample resources against their JSON schemas.

.PHONY: validate-star
validate-star: schemas-star ## Validate star sample resources against their JSON schemas.
	@kubeconform -summary \
		-schema-location default \
		-schema-location "$(STAR_SCHEMA_DIR)/{{.Group}}_{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
		$(STAR_SAMPLE_DIR)

.PHONY: validate-galaxy
validate-galaxy: schemas-galaxy ## Validate galaxy sample resources against their JSON schemas.
	@kubeconform -summary \
		-schema-location default \
		-schema-location "$(GALAXY_SCHEMA_DIR)/{{.Group}}_{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
		$(GALAXY_SAMPLE_DIR)

.PHONY: helm-lint
helm-lint: helm-lint-star helm-lint-galaxy ## Run helm lint against all charts.

.PHONY: helm-lint-star
helm-lint-star: hermit ## Run helm lint against the star chart.
	$(HELM) lint charts/star

.PHONY: helm-lint-galaxy
helm-lint-galaxy: hermit ## Run helm lint against the galaxy chart.
	$(HELM) lint charts/galaxy

## Tool Binaries (Testing)
GINKGO ?= $(LOCALBIN)/ginkgo

##@ Testing

.PHONY: test
test: hermit test-star test-galaxy test-pkg ## Run all unit tests.

.PHONY: test-star
test-star: hermit manifests generate-star fmt vet setup-envtest ginkgo ## Run unit tests for the star operator only.
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		$(GINKGO) --coverprofile=cover-star.out -v ./internal/star/...

.PHONY: test-galaxy
test-galaxy: hermit manifests generate-galaxy fmt vet setup-envtest ginkgo ## Run unit tests for the galaxy operator only.
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		$(GINKGO) --coverprofile=cover-galaxy.out -v ./internal/galaxy/...

.PHONY: test-pkg
test-pkg: hermit fmt vet ginkgo ## Run unit tests for shared pkg packages.
	$(GINKGO) --coverprofile=cover-pkg.out -v ./pkg/...

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
build: manifests generate fmt vet build-star build-galaxy ## Build all operator binaries.

.PHONY: build-star
build-star: hermit ## Build the star operator binary.
	go build -o bin/star ./cmd/star/main.go

.PHONY: build-galaxy
build-galaxy: hermit ## Build the galaxy operator binary.
	go build -o bin/galaxy ./cmd/galaxy/main.go

.PHONY: run-star
run-star: manifests-star generate-star fmt vet ## Run the star operator from your host.
	go run ./cmd/star/main.go

.PHONY: run-galaxy
run-galaxy: manifests-galaxy generate-galaxy fmt vet ## Run the galaxy operator from your host.
	go run ./cmd/galaxy/main.go

# These targets are only used for local environments (not in pipeline)
.PHONY: docker-build
docker-build: docker-build-star docker-build-galaxy ## Build container images for all operators (local use only).

.PHONY: docker-build-star
docker-build-star: hermit ## Build the star operator container image (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile --build-arg OPERATOR_NAME=star -t $(STAR_IMAGE) .

.PHONY: docker-build-galaxy
docker-build-galaxy: hermit ## Build the galaxy operator container image (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile --build-arg OPERATOR_NAME=galaxy -t $(GALAXY_IMAGE) .

.PHONY: docker-bake
docker-bake: hermit ## Build all container images using docker buildx bake (multi-platform, CI-compatible).
	$(CONTAINER_TOOL) buildx bake --file docker-bake.hcl

.PHONY: docker-push
docker-push: docker-push-star docker-push-galaxy ## Push container images for all operators.

.PHONY: docker-push-star
docker-push-star: ## Push the star operator container image.
	$(CONTAINER_TOOL) push $(STAR_IMAGE)

.PHONY: docker-push-galaxy
docker-push-galaxy: ## Push the galaxy operator container image.
	$(CONTAINER_TOOL) push $(GALAXY_IMAGE)

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
