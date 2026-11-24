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
  --target-setup ./conformance/target-secret.yaml \
  --report-dir ./conformance-results
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

The suite's `BeforeSuite` hook creates registry Secrets in a generated, run-scoped namespace and
passes their names and namespace to the existing
`pkg/ocm/credentials.ResolverFromRefs` function. That function fetches the Secrets
from Kubernetes and builds the credential resolver. Registry Secrets are separate
from per-case target setup resources; a Konfidence project hierarchy is not required.

The hook registers namespace deletion with Ginkgo's `DeferCleanup` as soon as the
namespace is created. Cleanup runs after execution, including setup failures and
interruption. Kubernetes removes the resources in that namespace
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

## Results

Scenarios are structured with Ginkgo (`Describe`, `Context`, `It`) and use Gomega
for assertions and asynchronous checks. The runner and scenarios are compiled
into `kden-conformance`; executing a release binary requires neither Go nor the
Ginkgo CLI.

Each suite invocation writes framework reports to `--report-dir`, which defaults
to `./conformance-results`:

```text
conformance-results/
  junit.xml       JUnit XML for CI test-result integration
  report.json     Ginkgo JSON with scenario states, failures and execution details
```

Ginkgo owns execution, lifecycle hooks, timeout handling and report generation.
Its native JUnit and JSON reporters write results after cleanup, including for failed
checks and suite setup or cleanup failures. Failed checks, lifecycle failures or
report write errors produce a non-zero command exit status. Argument parsing and
local input validation happen in Cobra before Ginkgo starts and do not generate
suite reports.
An invocation with no executed scenarios does not establish deployer conformance.
Following Ginkgo's lifecycle rules, suite setup and cleanup run only when there
are runnable specs.

These files are results for the tested deployer and environment, not Konfidence
release assets. Use a separate report directory for each invocation to preserve
results; subsequent runs overwrite the same filenames. Treat reports as potentially
sensitive because assertion output and scenario logs may contain environment data.

For example, upload the results from a deployer's GitHub Actions workflow even
when the conformance step fails:

```yaml
- name: Upload conformance results
  if: always()
  uses: actions/upload-artifact@v7
  with:
    name: conformance-results
    path: conformance-results/
```

## Development

### DeploymentTarget scenarios

The initial section 2 checks live in
`internal/conformance/suite/deploymenttarget.go`. Each covered keyword has its own
Ginkgo `It` and keyword label:

| Keyword | Scenario |
| --- | --- |
| [0018] | Supported target becomes ready; foreign types receive no observable spec, status, finalizer, label, annotation, owner-reference, or deletion-timestamp changes. |
| [0019] | Foreign targets' status remains unchanged. |
| [0022] | A target referencing a nonexistent parameter resource is rejected with `Ready=False`. |
| [0027] | The supplied valid target becomes `Ready=True`. |
| [0029] | A missing parameter resource produces `Ready=False` with a nonempty reason and message. |

The shared hook in `scenario_setup.go` configures timeouts and polling,
initializes the client, creates a fresh namespace, and copies setup resources
before any target is created. `DeferCleanup` requests namespace deletion even
after a failure, partial setup, or interruption. Cluster-scoped setup resources
are rejected. The suite never modifies the caller's baseline manifests.

Scenario files are separated by responsibility:

- `scenario_setup.go`: common fixture hooks, timeout configuration and budgets,
  client initialization, namespace lifecycle, and setup-resource creation.
- `deploymenttarget_setup.go`: baseline target construction, missing-reference
  variants, foreign target fixtures, and target API access.
- `deploymenttarget.go`: clearly marked context-specific setup hooks followed by
  keyword specs and contract assertions. `It` bodies assert behavior; fixture
  creation belongs in the preceding `BeforeEach` hooks.

Gomega's asynchronous defaults are installed by the common hook, including
enforcing the status timeout when assertions receive a `SpecContext`.

For ownership checks, a ready supported target demonstrates an active deployer.
The suite then creates two randomly generated foreign types: one under the same
provider domain and one under a different domain. Gomega `Consistently` checks
them throughout the observation interval. Run against an isolated environment
with no other controller claiming these generated types. The checks detect
observable reconciliation effects, not internal reconcile method invocations.

The missing-reference scenarios are derived by changing only the referenced
resource's name. They do not guess how to corrupt provider-owned payloads.
Checks [0022] and [0029] are skipped when the baseline has no parameter reference.
For [0029], descriptive quality beyond a nonempty reason/message still requires
human judgment.

Configure positive Go durations through:

- `KDEN_CONFORMANCE_TIMEOUT` (default `1m`): per-scenario setup and status waits.
- `KDEN_CONFORMANCE_OBSERVATION_TIMEOUT` (default `10s`): foreign-target observations.

Status assertions use `Eventually`; ownership observations use `Consistently`,
both polling every 250 ms. Ginkgo `SpecTimeout` bounds each complete scenario,
including setup. These execution bounds are not normative deployer deadlines.

The initial set does not create `ArtifactDeployment` resources, resolve the
application artifact, or inspect destination runtime resources. The remaining
section 2 keywords are deliberately excluded:

- [0020–0021], [0028]: optional, provider-specific behavior.
- [0023], [0032–0033]: need independent destination-runtime observations.
- [0024–0025]: need documentation review or provider-specific safety fixtures.
- [0026]: needs a meaningful provider-specific referenced-resource mutation.
- [0030], [0034]: concern core rather than the deployer.
- [0031]: presence alone cannot verify a name and release version without an
  agreed controller-identifier format.

`deploymenttarget_test.go` exercises these scenarios through the existing
framework in subprocesses with a fake client. It checks native JUnit/JSON
results, failures for each covered keyword, skips, delayed foreign-target changes,
setup ordering, input preservation, and cleanup after failure and cancellation.
These implementation tests do not establish a real deployer's conformance.

The implementation follows the repository's CLI layout:

```text
cmd/kden-conformance/          process entry point, signals, exit status
  cmd/                        Cobra flags and presentation
internal/conformance/
  setup/                      local input validation and credential loading
  suite/                      Ginkgo runner, scenario definitions and reports
conformance/specs/            normative deployer contract
```

Build locally with `make build-conformance`, or run from source using
`go run ./cmd/kden-conformance run` with the same arguments.

Run `make test-conformance-impl` (or `go test ./internal/conformance/... ./cmd/kden-conformance/...`)
to test the conformance tool implementation, not to execute deployer conformance
scenarios. The regular lint/test/build workflow gates builds on these implementation tests.

Define conformance scenarios in regular `.go` files under
`internal/conformance/suite`, using the `Environment` passed to
`registerScenarios`. The enclosing `Describe` registers `registerScenarioSetup`
once; new scenario groups reuse its namespace and timing configuration.
Use Ginkgo lifecycle hooks for resource-specific setup and cleanup,
and Gomega assertions for the contract checks. Use Ginkgo's `SpecContext`,
`SpecTimeout` and `NodeTimeout` for interruptible operations. The suite and
reporter configurations start from Ginkgo's defaults, with report destinations
selected by `--report-dir`. Keep implementation tests in
`_test.go` files; they are not included in the release binary.

Ginkgo supports one suite execution per process. Run a new `kden-conformance`
process for each artifact/target pair.

## Releases

The existing core release workflow publishes this executable through
`.goreleaser.yaml`; no separate version stream or publishing workflow is needed.
Assets are `kden-conformance-{linux,darwin,windows}-{x86_64,arm64}` archives
(`.tar.gz`, or `.zip` on Windows), accompanied by the release's `checksums.txt`.
The binary's version is injected from the same core release tag.
