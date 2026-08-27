# Image registry and tag used by all build/push targets
REGISTRY ?= ghcr.io/konfidence-project
TAG      ?= dev

# Namespace for deploying to Kubernetes cluster
NAMESPACE ?= konfidence-system

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

# Internal controller packages of the konfidence operator, input to manifest
# (RBAC) generation. Auto-discover by finding all internal/ subdirs containing
# setup.go, then append /internal/controller.
OPERATOR_INTERNAL_DIRS = $(shell find internal -maxdepth 2 -name setup.go -exec dirname {} \; | sed 's|^|./|; s|$$|/internal/controller|' | sort)
OPERATOR_INTERNAL_PATHS = $(foreach d,$(OPERATOR_INTERNAL_DIRS),paths="$(d)")

# Ginkgo suites of the operator domains, input to test-operators. Discovered
# separately from the controller packages above: manifests must scan every
# controller dir, tests must run every suite dir (which may sit outside
# internal/controller, e.g. domain roots or internal/promotion).
OPERATOR_SUITE_DIRS = $(shell find internal -maxdepth 2 -name setup.go -exec dirname {} \; | xargs -I{} find {} -name "*suite_test.go" | xargs -n1 dirname | sort -u | sed 's|^|./|')

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
OAPI_CODEGEN   ?= go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0

## Image names
OPERATOR_IMAGE = $(REGISTRY)/konfidence-operator:$(TAG)

## Local API server configuration
API_OIDC_ENABLED ?= true
API_OIDC_ISSUER_URL ?= http://localhost:5556/oidc
API_OIDC_CLIENT_SECRET ?= konfidence-local-secret
API_OIDC_SCOPES ?= openid,profile,email,groups
API_OIDC_REDIRECT_URL ?= http://localhost:8090/api/v1/auth/callback
API_OIDC_ALLOW_RETURN_URLS ?=
API_SESSION_STORAGE_TYPE ?= in-memory

## No-auth mode configuration (for UI development without IDP)
API_NOAUTH_RETURN_URL ?= http://localhost:5173

.PHONY: all
all: api build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: hermit manifests-crds ## Generate CRDs, RBAC, and webhook manifests for the konfidence operator chart.
	@echo "Generating manifests for konfidence..."
	@mkdir -p $(CRD_DIR) charts/konfidence/templates/crds config/rbac config/webhook
	$(CONTROLLER_GEN) rbac:roleName=konfidence-manager \
		$(OPERATOR_INTERNAL_PATHS) \
		$(API_PATHS) \
		output:rbac:artifacts:config=config/rbac
	$(CONTROLLER_GEN) webhook $(API_PATHS) output:webhook:artifacts:config=config/webhook
	@rm -f $(CRD_DIR)/*.yaml charts/konfidence/templates/crds/*.yaml
	@cp $(CRD_STAGING_DIR)/*.yaml $(CRD_DIR)/
	for f in $(CRD_DIR)/*.yaml; do \
		charts/patch-crd.sh konfidence "$$f" "charts/konfidence/templates/crds/$$(basename $$f)"; \
	done
	charts/patch-clusterrole.sh konfidence config/rbac/role.yaml charts/konfidence/templates/clusterrole.yaml
	charts/patch-webhook.sh konfidence config/webhook charts/konfidence/templates/validatingwebhookconfiguration.yaml
	$(HELM_DOCS) -c charts/konfidence > charts/konfidence/README.md

.PHONY: manifests-crds
manifests-crds: hermit ## Generate the full CRD set (single konfidence.cloud group) into the staging dir.
	@echo "Generating CRDs for the konfidence.cloud group..."
	@rm -rf $(CRD_STAGING_DIR)
	@mkdir -p $(CRD_STAGING_DIR)
	$(CONTROLLER_GEN) crd $(API_PATHS) output:crd:artifacts:config=$(CRD_STAGING_DIR)

.PHONY: generate
generate: hermit ## Generate DeepCopy implementations.
	$(CONTROLLER_GEN) object $(API_PATHS)

.PHONY: docs
docs: docs-reference docs-cli ## Regenerate all API references (CRD + CLI).

.PHONY: docs-cli
docs-cli: hermit ## Generate kden CLI reference into api/docs/cli.md.
	@mkdir -p api/docs
	go run ./cmd/kden docs --type markdown --dir api/docs --frontmatter

.PHONY: check-generate
check-generate: docs ## Verify all API references in api/docs/ are committed and up to date.
	@hack/check-generate.sh

.PHONY: check-manifests
check-manifests: manifests ## Verify committed manifests (CRDs, RBAC, webhook) are in sync with Go types.
	@hack/check-manifests.sh


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

.PHONY: webhook-certs
webhook-certs: ## Generate self-signed certificates for local webhook development.
	@./hack/generate-webhook-certs.sh

##@ API

.PHONY: api
api: hermit manifests generate generate-api schemas docs-reference helm-lint ## Run full API generation pipeline (manifests, deepcopy, OpenAPI clients/server, schemas, CRD reference doc, helm lint).

.PHONY: generate-api
generate-api: hermit ## Generate the OpenAPI server and kden API client from api/openapi.yaml.
	$(OAPI_CODEGEN) -config api/codegen-server.yaml api/openapi.yaml
	$(OAPI_CODEGEN) -config api/codegen-client.yaml api/openapi.yaml

.PHONY: docs-reference
docs-reference: hermit ## Generate + transform the CRD reference into api/docs/crd.md.
	@echo "Generating CRD reference (api/docs/crd.md)..."
	@mkdir -p api/docs
	@set -e; \
	tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	crd-ref-docs \
		--source-path="api/v1alpha1" \
		--config="$(REPO_ROOT)/api/.crd-ref-docs.config.yaml" \
		--renderer=markdown \
		--output-path="$$tmp"; \
	hack/transform-crd-docs.sh < "$$tmp/out.md" > "$$tmp/crd.md"; \
	if ! grep -q '^### Resource Types' "$$tmp/crd.md"; then \
		echo "::error::transform produced a degenerate crd.md (no 'Resource Types' section) — refusing to overwrite."; \
		exit 1; \
	fi; \
	mv "$$tmp/crd.md" api/docs/crd.md

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
validate: schemas check-samples ## Validate all sample resources against their JSON schemas.
	@kubeconform -summary \
		-schema-location default \
		-schema-location "$(SCHEMA_DIR)/{{.Group}}_{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
		$(SAMPLE_DIR)

.PHONY: check-samples
check-samples: hermit manifests ## Verify every top-level CRD Kind has a sample in $(SAMPLE_DIR).
	@SAMPLE_DIR=$(SAMPLE_DIR) CRD_DIR=$(CRD_DIR) hack/check-samples.sh

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
		$(GINKGO) --coverprofile=cover-operators.out -v $(OPERATOR_SUITE_DIRS) ./cmd/konfidence/...

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
	go run ./cmd/api/main.go \
		--oidc-enabled=$(API_OIDC_ENABLED) \
		--oidc-issuer-url=$(API_OIDC_ISSUER_URL) \
		--oidc-client-secret=$(API_OIDC_CLIENT_SECRET) \
		--oidc-scopes=$(API_OIDC_SCOPES) \
		--oidc-redirect-url=$(API_OIDC_REDIRECT_URL) \
		--oidc-allow-return-urls=$(API_OIDC_ALLOW_RETURN_URLS) \
		--session-storage-type=$(API_SESSION_STORAGE_TYPE) \

.PHONY: run-kden-api-noauth
run-kden-api-noauth: fmt vet ## Run the kden API server with auth disabled (static admin user).
	@echo "Starting API server with authentication DISABLED"
	@echo "Static admin user: Local Admin <admin@local>"
	@echo "All projects will be accessible with admin role"
	@echo ""
	go run ./cmd/api/main.go \
		--oidc-enabled=false \
		--oidc-allow-return-urls=$(API_NOAUTH_RETURN_URL) \
		--session-storage-type=in-memory \
		--session-cookie-secure=false

.PHONY: stop-kden-api
stop-kden-api: ## Stop any running kden API server instances.
	@echo "Stopping kden API server..."
	@pkill -f "go-build.*api" 2>/dev/null && echo "[OK] API server stopped" || true
	@pkill -f "cmd/api" 2>/dev/null || true
	@lsof -ti:8090 | xargs kill -9 2>/dev/null && echo "[OK] Killed process on port 8090" || echo "[OK] No process running on port 8090"

.PHONY: test-kden-api-noauth
test-kden-api-noauth: ## Test the no-auth API setup (requires API running with make run-kden-api-noauth).
	@echo "Testing API server with authentication disabled..."
	@echo ""
	@echo "1. Testing /healthz endpoint..."
	@curl -sf http://localhost:8090/healthz > /dev/null && echo "   [OK] Health check passed" || (echo "   [FAIL] Health check failed - is the API running?" && exit 1)
	@echo ""
	@echo "2. Testing /api/v1/identity endpoint (static admin user)..."
	@IDENTITY=$$(curl -sf http://localhost:8090/api/v1/identity) && \
		echo "   Response: $$IDENTITY" && \
		echo "$$IDENTITY" | grep -q "admin@local" && echo "   [OK] Static admin user verified" || (echo "   [FAIL] Unexpected identity response" && exit 1)
	@echo ""
	@echo "3. Testing /api/v1/login redirect (should redirect to return URL)..."
	@LOCATION=$$(curl -sf -o /dev/null -w '%{redirect_url}' "http://localhost:8090/api/v1/login?return_url=$(API_NOAUTH_RETURN_URL)") && \
		echo "   Redirect: $$LOCATION" && \
		[ "$$LOCATION" = "$(API_NOAUTH_RETURN_URL)" ] && echo "   [OK] Login redirects correctly without IDP" || (echo "   [FAIL] Unexpected redirect" && exit 1)
	@echo ""
	@echo "4. Testing /api/v1/projects endpoint (admin access)..."
	@PROJECTS=$$(curl -sf http://localhost:8090/api/v1/projects) && \
		echo "   Response: $$PROJECTS" && \
		echo "   [OK] Projects endpoint accessible"
	@echo ""
	@echo "5. Testing /api/v1/logout endpoint..."
	@curl -sf -X POST http://localhost:8090/api/v1/logout > /dev/null && echo "   [OK] Logout endpoint works" || echo "   [WARN] Logout returned non-200 (expected in no-auth mode)"
	@echo ""
	@echo "==================================================================="
	@echo "All tests passed! The API is ready for UI development."
	@echo ""
	@echo "UI developers can:"
	@echo "  - Call /api/v1/identity to get the static admin user"
	@echo "  - Use /api/v1/login which redirects directly to return_url"
	@echo "  - Access all project APIs with admin privileges"
	@echo "==================================================================="

# These targets are only used for local environments (not in pipeline)
.PHONY: docker-build
docker-build: hermit ## Build the konfidence operator container image (local use only).
	$(CONTAINER_TOOL) build -f Dockerfile --build-arg TARGETPLATFORM=bin --build-arg OPERATOR_NAME=konfidence -t $(OPERATOR_IMAGE) .

.PHONY: docker-push
docker-push: ## Push the konfidence operator container image.
	$(CONTAINER_TOOL) push $(OPERATOR_IMAGE)

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: hermit manifests ## Install CRDs into the cluster specified in ~/.kube/config.
	$(HELM) upgrade --install konfidence charts/konfidence --set controller.install=false --set api.enabled=false --set crd.keep=false

.PHONY: uninstall
uninstall: hermit ## Uninstall CRDs from the cluster. Use ignore-not-found=true to suppress errors.
	$(HELM) uninstall konfidence --ignore-not-found

.PHONY: deploy
deploy: hermit manifests ## Deploy the konfidence operator to the cluster specified in ~/.kube/config.
	@# Ensure namespace exists
	@echo "Checking if namespace '$(NAMESPACE)' exists..."
	@kubectl get namespace $(NAMESPACE) >/dev/null 2>&1 || \
		(echo "Creating namespace '$(NAMESPACE)'..." && kubectl create namespace $(NAMESPACE))
	@# Check for webhook certificates and prepare CA bundle
	@HELM_EXTRA_ARGS=""; \
	if [ -f /tmp/k8s-webhook-server/serving-certs/tls.crt ] && [ -f /tmp/k8s-webhook-server/serving-certs/tls.key ]; then \
		echo "Creating webhook certificate secret in namespace '$(NAMESPACE)'..."; \
		kubectl create secret tls konfidence-webhook-server-cert \
			--cert=/tmp/k8s-webhook-server/serving-certs/tls.crt \
			--key=/tmp/k8s-webhook-server/serving-certs/tls.key \
			--namespace=$(NAMESPACE) \
			--dry-run=client -o yaml | kubectl apply -f - || true; \
		echo "✓ Webhook certificate secret created/updated"; \
		if [ -f /tmp/k8s-webhook-server/serving-certs/ca.crt ]; then \
			echo "Setting webhook CA bundle from ca.crt..."; \
			CA_BUNDLE=$$(base64 < /tmp/k8s-webhook-server/serving-certs/ca.crt | tr -d '\n'); \
			HELM_EXTRA_ARGS="--set-string webhook.caBundle=$$CA_BUNDLE"; \
		fi; \
	elif grep -q "webhook.enabled.*true" charts/konfidence/values.yaml 2>/dev/null; then \
		echo ""; \
		echo "⚠️  Warning: Webhooks are enabled but certificates not found."; \
		echo ""; \
		echo "Generate certificates first by running:"; \
		echo "  make webhook-certs"; \
		echo ""; \
		echo "Or disable webhooks with:"; \
		echo "  --set webhook.enabled=false"; \
		echo ""; \
		read -p "Continue deployment without webhook certificates? (y/N): " -n 1 -r CONTINUE; \
		echo ""; \
		if [[ ! $$CONTINUE =~ ^[Yy]$$ ]]; then \
			echo "Deployment cancelled."; \
			exit 1; \
		fi; \
	fi; \
	$(HELM) upgrade --install konfidence charts/konfidence \
		--namespace=$(NAMESPACE) \
		--set image.repository=$(REGISTRY)/konfidence-operator \
		--set image.tag=$(TAG) \
		--set api.image.repository=$(REGISTRY)/api \
		--set api.image.tag=$(TAG) \
		--set crd.keep=false \
		$$HELM_EXTRA_ARGS

.PHONY: undeploy
undeploy: hermit ## Undeploy the konfidence operator. Use ignore-not-found=true to suppress errors.
	$(HELM) uninstall konfidence \
		--namespace=$(NAMESPACE) \
		--ignore-not-found

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
