# Deployer conformance

The deployer conformance suite is intended to help deployer authors verify their
implementation against the [Konfidence deployer specification](specs/deployer.md).
It provides a common set of checks for the behavior of `DeploymentTarget` and
`ArtifactDeployment` resources, independently of the deployer's target runtime.

The suite is a standalone executable, `kden-conformance`. Its scope is the deployer
contract: running a complete Konfidence control plane is not a prerequisite.
The specification defines the requirements; the suite provides repeatable checks
for those requirements.

## Test environment

Deployer authors prepare:

- A Kubernetes test cluster with the Konfidence CRDs matching the suite version.
- The deployer under test and its runtime dependencies.
- A small, deterministic OCM application artifact for the supported artifact type.
- A deployment-target spec describing how to access the target runtime.
- Any target-specific configuration or credentials needed by the deployer.

Each invocation covers one artifact type and one artifact/target pair. For a
deployer supporting multiple artifact types, run the suite separately for each type.

## Test inputs and derived scenarios

The deployer author supplies a valid baseline consisting of:

- **An application artifact:** an OCM component version containing the component
  descriptor, artifact manifests and deployer-specific resources needed for a
  successful deployment, supplied through an OCI image reference.
- **A deployment-target spec:** the deployer configuration for that artifact type,
  including references to any required target configuration or credentials.
- **Optional target setup resources:** Kubernetes manifests for the resources
  referenced by the target configuration, such as Secrets or ConfigMaps.

All test cases are derived from these baseline inputs. The suite uses the valid
artifact and target configuration to construct the resources under test and
derive the variants needed to check the deployer contract. Deployer authors do
not need to author separate inputs or Kubernetes resources for each test case.
Provider-specific payloads remain deployer-owned; the suite does not guess how
to mutate them.

Namespaces, shared and per-case scopes, resource ordering, isolation and cleanup
are responsibilities of the suite during test execution, not part of the
developer-supplied test setup.

## Usage

```sh
kden-conformance run \
  --kubeconfig ./conformance.kubeconfig \
  --artifact-reference ghcr.io/example/conformance-app:1.0.0 \
  --deployment-target-spec ./conformance/deployment-target-spec.yaml \
  --target-setup ./conformance/target-secret.yaml
```

Use `kden-conformance run --help` for execution flags. Invoking
`kden-conformance` without a subcommand shows help, without executing the suite.
`kden-conformance version`
prints the release version and build metadata as JSON without requiring test inputs.

### Inputs

- `--kubeconfig` selects an explicit file; otherwise client-go uses `KUBECONFIG`
  or `~/.kube/config`. The kubeconfig's `current-context` selects the test cluster.
- `--artifact-reference` is an OCI **image** reference with a tag or digest,
  not an OCM `repository//component:version` reference.
- `--deployment-target-spec` accepts YAML or JSON containing only the fields
  below `DeploymentTarget.spec`, without resource metadata.
- `--target-setup` accepts optional, repeatable files containing namespaced
  Kubernetes resources such as Secrets or ConfigMaps. Multiple YAML documents
  are supported. Omit `metadata.namespace`; the suite assigns namespaces during
  execution. Supply individual resources rather than List wrappers.

Example target spec:

```yaml
artifactType: app.example.com
parameters:
  kind: Secret
  name: conformance-target-credentials
```

### Credentials

Cluster access, OCM registry access and target-runtime access are separate concerns:

- **Cluster access** uses the credentials and TLS settings in the kubeconfig.
- **OCM registry access** uses optional Kubernetes Secret manifests supplied through
  `--ocm-registry-config`. Multiple YAML documents are supported; both `data` and
  `stringData` are accepted. Secrets contain `.dockerconfigjson` or `.ocmconfig`
  configuration. Omit `metadata.namespace` so the suite can assign its shared
  namespace. Without this input, registry access is anonymous; local Docker
  configuration is not loaded implicitly.
- **Target-runtime access** uses resources supplied through `--target-setup`,
  referenced by the deployment-target parameters.

The suite creates the registry Secrets in a generated, run-scoped namespace and
passes their names and namespace to the existing
`pkg/ocm/credentials.ResolverFromRefs` function. That function fetches the Secrets
from Kubernetes and builds the credential resolver. Registry Secrets are separate
from per-case target setup resources; a Konfidence project hierarchy is not required.

After execution, including setup failures and cancellation, the suite requests
deletion of its shared namespace. Kubernetes removes the resources in that namespace
asynchronously. The kubeconfig must permit namespace creation/deletion and Secret
creation/reading. The suite does not overwrite existing developer-owned Secrets.

For example, `--ocm-registry-config ./registry-secrets.yaml` can load:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: conformance-registry
type: kubernetes.io/dockerconfigjson
stringData:
  .dockerconfigjson: |
    {"auths":{"registry.example.com":{"auth":"<base64-encoded-username:password>"}}}
```

Use only trusted configuration files: kubeconfig exec authentication can execute
external programs. Treat all manifests containing Secrets as sensitive input.

## Development

The implementation follows the repository's CLI layout:

```text
cmd/kden-conformance/          process entry point, signals, exit status
  cmd/                        Cobra flags and presentation
internal/conformance/
  setup/                      local input validation and credential loading
  suite/                      conformance execution and results
conformance/specs/            normative deployer contract
```

Build locally with `make build-conformance`, or run from source using
`go run ./cmd/kden-conformance run` with the same arguments.

Run `make test-conformance-impl` (or `go test ./internal/conformance/... ./cmd/kden-conformance/...`)
to test the conformance tool implementation, not to execute deployer conformance
scenarios. The regular lint/test/build workflow gates builds on these implementation tests.

## Releases

The existing core release workflow publishes this executable through
`.goreleaser.yaml`; no separate version stream or publishing workflow is needed.
Assets are `kden-conformance-{linux,darwin,windows}-{x86_64,arm64}` archives
(`.tar.gz`, or `.zip` on Windows), accompanied by the release's `checksums.txt`.
The binary's version is injected from the same core release tag.
