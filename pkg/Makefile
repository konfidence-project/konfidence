GINKGO := $(GOPATH)/bin/ginkgo
MOCKGEN := $(GOPATH)/bin/mockgen

.PHONY: install-tools
install-tools:
	go install go.uber.org/mock/mockgen@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@latest

.PHONY: generate
generate: ## Generate code.
	go generate -v ./...

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: install-tools generate fmt vet
	$(GINKGO) -v -r