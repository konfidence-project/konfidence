# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: generate lint test

.PHONY: $(MOCKGEN) generate
generate: mockgen ## Generate code.
	go generate -v ./...

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: generate ginkgo fmt vet
	$(GINKGO) -v -r

.PHONY: golangci-lint lint
lint: ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: golangci-lint lint-fix
lint-fix: ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: golangci-lint lint-config
lint-config: ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

#######################
# Tools

GINKGO ?= $(GOBIN)/ginkgo
GINKGO_VERSION ?= latest

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary.
$(GINKGO):
	$(call go-install-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo,$(GINKGO_VERSION))

MOCKGEN ?= $(GOBIN)/mockgen
MOCKGEN_VERSION ?= latest

.PHONY: mockgen
mockgen: $(MOCKGEN) ## Download mockgen locally if necessary.
$(MOCKGEN):
	$(call go-install-tool,$(MOCKGEN),go.uber.org/mock/mockgen,$(MOCKGEN_VERSION))

GOLANGCI_LINT ?= $(GOBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.2.2

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT):
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
