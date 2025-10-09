[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/crds)](https://api.reuse.software/info/github.com/konfidence-project/crds)

# Konfidence CRDs

## About this project

This repository provides a set of Custom Resource Definitions (CRDs) which are required to run Konfidence.
It is built using the Kubebuilder framework and make use of the `controller-gen` CLI.

### Module Structure
```
api/
├── common/v1alpha1/     # Common CRDs (Stage)
└── landscape/v1alpha1/  # Landscape CRDs (Deployment, Execution, Vector management)
```

### API Documentation
- [Common APIs](api/common/docs/README.md) - Stage definitions
- [Landscape APIs](api/landscape/docs/README.md) - Deployment and execution resources

## Development

### How to update CRDs

1. Implement your changes in the Go types of your module, e.g. `api/common/v1alpha1/stage_types.go`.
Make sure to follow the [kubebuilder conventions](https://book.kubebuilder.io/reference/markers/crd-validation.html) for defining CRD fields validation.
Always add a comment to each field/struct to describe its purpose and to generate meaningful documentation.
2. Update the examples for the CRDs that you changed, e.g. `api/common/v1alpha1/config/samples/stage.yaml`.
3. Run `make all` to update the generated code and run schema validations.
4. Commit and push your changes.


## How to use

### Installation

Apply the CRDs to your Kubernetes cluster using kustomize:
```bash
kubectl apply -f api/common/config/release
kubectl apply -f api/landscape/config/release
```

### Integration with Kubebuilder Projects

To use these CRDs in your Kubebuilder controller project, first add the dependencies:

```bash
go get github.com/konfidence-project/crds/api/common
go get github.com/konfidence-project/crds/api/landscape
```

Then you can run controller-gen directly:

```bash
# Generate Konfidence Common CRDs
controller-gen crd paths="github.com/konfidence-project/crds/api/common/..." output:crd:artifacts:config=config/crd/common

# Generate Konfidence Landscape CRDs
controller-gen crd paths="github.com/konfidence-project/crds/api/landscape/..." output:crd:artifacts:config=config/crd/landscape
```

Or add the following target to your Makefile:

```makefile
.PHONY: manifests
manifests: controller-gen ## Generate ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) crd paths="github.com/konfidence-project/crds/api/common/..." output:crd:artifacts:config=config/crd/common
	$(CONTROLLER_GEN) crd paths="github.com/konfidence-project/crds/api/landscape/..." output:crd:artifacts:config=config/crd/landscape
```

This will:
- Generate your project's own CRDs and RBAC in `config/crd/bases/`
- Generate Konfidence Common CRDs in `config/crd/common/`
- Generate Konfidence Landscape CRDs in `config/crd/landscape/`

Then reference them in your `config/default/kustomization.yaml`:
```yaml
resources:
- ../crd/bases
- ../crd/common
- ../crd/landscape
```

### Development

#### Generate CRDs and code
```bash
make all          # Generate manifests, code, format, lint, validate, and docs
make manifests    # Generate CRD manifests only
make generate     # Generate Go code (deepcopy, etc.)
```

#### Validation and quality
```bash
make fmt          # Format Go code
make vet          # Run go vet
make lint         # Run golangci-lint
```

#### Documentation
```bash
make docs         # Generate CRD reference documentation
```

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/konfidence-project/<your-project>/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/konfidence-project/<your-project>/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and crds contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/konfidence-project/<your-project>).
