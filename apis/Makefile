# Root Makefile - Proxy for all go sub-modules
# Runs targets across all modules for better developer experience

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

MODULES := $(shell find . -name "go.mod" -not -path "./vendor/*" -exec dirname {} \;)
REPO_ROOT := $(shell git rev-parse --show-toplevel)

# Generic template for module targets
define make_target
.PHONY: $(1)
$(1): ## $(2)
	@echo "$(2)"
	@for module in $(MODULES); do \
		$(MAKE) -C $$$$module $(1) || exit 1; \
	done
	@echo "$(1) completed for all modules"
endef

.PHONY: help
help: ## Display this help.
	@echo "Root Makefile - runs targets across all modules:"
	@echo "Modules: $(MODULES)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: all
all: manifests generate fmt vet lint docs schemas validate ## Run all code generators and checks across all modules

# Generate all standard targets
$(eval $(call make_target,fmt,Run go fmt across all modules))
$(eval $(call make_target,vet,Run go vet across all modules))
$(eval $(call make_target,lint,Run golangci-lint across all modules))
$(eval $(call make_target,lint-fix,Run golangci-lint with fixes across all modules))
$(eval $(call make_target,test,Run tests across all modules))
$(eval $(call make_target,generate,Generate code across all modules))
$(eval $(call make_target,manifests,Generate manifests across all modules))
$(eval $(call make_target,install,Install CRDs into the K8s cluster specified in kubeconfig))
$(eval $(call make_target,build,Build all modules))
$(eval $(call make_target,docs,Generate CRD documentation across all modules))
$(eval $(call make_target,schemas,Extract JSON schemas for each CRD version))
$(eval $(call make_target,validate,Validate sample resources against their JSON schemas))

##@ Utilities

.PHONY: modules
modules: ## List all modules
	@echo "Modules:"
	@for module in $(MODULES); do \
		echo "  $$module"; \
	done

.PHONY: clean
clean: ## Clean all modules
	@echo "Cleaning all modules..."
	@for module in $(MODULES); do \
		echo "  $$module"; \
		rm -rf $$module/cover.out; \
	done
	@rm -rf .bin
	@echo "clean completed for all modules"

.PHONY: hermit
hermit: ## Check if Hermit is installed and its environment is activated.
	@command -v hermit >/dev/null 2>&1 || { \
		echo "Hermit is not installed. Please install it from https://cashapp.github.io/hermit/"; \
		exit 1; \
	}
	@hermit status >/dev/null 2>&1 || { \
		echo "Hermit environment is not activated. Run 'source ./bin/activate-hermit' or 'eval \"$$(hermit env)\"'"; \
		exit 1; \
	}

.PHONY: install-git-hooks
install-git-hooks: hermit  ## Install git hooks
	@echo "Setting up prek (pre-commit) installing git hooks..."
	prek install

.PHONY: uninstall-git-hooks
uninstall-git-hooks: hermit  ## Uninstall git hooks
	@echo "Uninstalling prek (pre-commit) git hooks..."
	prek uninstall