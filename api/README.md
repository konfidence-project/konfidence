[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/crds)](https://api.reuse.software/info/github.com/konfidence-project/crds)

# Konfidence CRDs

## About this project

This repository provides a set of Custom Resource Definitions (CRDs) which are required to run Konfidence.
It is built using the Kubebuilder framework and make use of the `controller-gen` CLI.

### Module Structure
```
api/
├── star/v1alpha1/    # Star CRDs (Deployment, Execution, Vector management)
└── galaxy/v1alpha1/  # Galaxy CRDs (StageConfiguration...)
```

### API Documentation
- [Star APIs](star/docs/README.md) - Stage definitions, deployment and execution resources
- [Galaxy APIs](galaxy/docs/README.md) - Galaxy resources

## Development

### How to update CRDs

1. Implement your changes in the Go types of your module, e.g. `api/star/v1alpha1/stage_types.go`.
Make sure to follow the [kubebuilder conventions](https://book.kubebuilder.io/reference/markers/crd-validation.html) for defining CRD fields validation.
Always add a comment to each field/struct to describe its purpose and to generate meaningful documentation.
2. Update the validation samples for the CRDs that you changed, e.g. `test/data/samples/star/landscape_v1alpha1_stageversion.yaml`.
3. Run `make all` to update the generated code and run schema validations.
4. Commit and push your changes.

## How to use

### Installation

Apply the CRDs to your Kubernetes cluster using the Helm charts:
```bash
helm upgrade --install star charts/star --set controller.install=false --set crd.keep=false
helm upgrade --install galaxy charts/galaxy --set controller.install=false --set crd.keep=false
```

The generated CRD manifests are maintained in the chart templates under `charts/star/templates/crds` and `charts/galaxy/templates/crds`.

### Integration with Kubebuilder Projects

To use these CRDs in your Kubebuilder controller project, first add the dependencies:

```bash
go get github.com/konfidence-project/konfidence/api/star
go get github.com/konfidence-project/konfidence/api/galaxy
```

Then you can run controller-gen directly:

```bash
# Generate Konfidence Star CRDs
controller-gen crd paths="github.com/konfidence-project/konfidence/api/star/..." output:crd:artifacts:config=config/crd/star

# Generate Konfidence Galaxy CRDs
controller-gen crd paths="github.com/konfidence-project/konfidence/api/galaxy/..." output:crd:artifacts:config=config/crd/galaxy
```

Or add the following target to your Makefile:

```makefile
.PHONY: manifests
manifests: controller-gen ## Generate ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) crd paths="github.com/konfidence-project/konfidence/api/star/..." output:crd:artifacts:config=config/crd/star
	$(CONTROLLER_GEN) crd paths="github.com/konfidence-project/konfidence/api/galaxy/..." output:crd:artifacts:config=config/crd/galaxy
```

This will generate your project's own CRDs and RBAC in `config/crd/bases/`, plus optional local copies of the Konfidence Star and Galaxy CRDs in `config/crd/star/` and `config/crd/galaxy/`.

### Development

#### Generate CRDs and code
```bash
make all          # Generate manifests, code, format, lint, validate, and docs
make manifests    # Generate CRD manifests only
make generate     # Generate Go code (deepcopy, etc.)
make api          # Generate CRDs, code, docs, and schemas
```

#### Validation and quality
```bash
make fmt          # Format Go code
make vet          # Run go vet
make lint         # Run golangci-lint
```

#### Documentation
```bash
make docs-star    # Generate Star CRD reference documentation
make docs-galaxy  # Generate Galaxy CRD reference documentation
```

#### Setup Git hooks

We use git hooks to check the conventional-commit formatting at "commit-msg".

```bash
make install-git-hooks    # install all git hooks with prek
make uninstall-git-hooks  # uninstall all git hooks with prek
```

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/konfidence-project/<your-project>/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/konfidence-project/<your-project>/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.
