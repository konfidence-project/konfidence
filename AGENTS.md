# Konfidence

## Coding Philosophy

- Code MUST follow the existing style and conventions already present in the
  file/package being touched. Do not introduce a competing style even if you
  personally prefer it.
- Code MUST comply with SOLID principles. A change that violates Single
  Responsibility, Open/Closed, Liskov Substitution, Interface Segregation, or
  Dependency Inversion is not done — it needs another pass.
- Cyclomatic complexity MUST be kept low. Prefer early returns, extracted
  functions, and flat control flow over deeply nested conditionals.
- Every change MUST come with tests. Untested code is not considered
  complete, regardless of how small the change looks.
- Tests MUST verify real business outcomes, not just execution. A test that
  only asserts a function was called, mocks away the behavior under test, or
  checks a trivial getter/setter is not a meaningful test and MUST NOT be
  used to satisfy this requirement. Prefer asserting on observable
  state/output that reflects what the code is actually supposed to
  accomplish.
- Exported functions, exported variables, and exported constants MUST carry a
  code documentation comment. Keep these to a minimum — state what is
  non-obvious, not a restatement of the identifier name. Wording MUST follow
  the guidelines in
  [konfidence-docs styleguide](https://github.com/konfidence-project/konfidence-docs/blob/main/src/docs/extend-customize/styleguide.md).
- Linting MUST pass. A change that does not pass lint is not done.
- Documentation is a living artifact, not a one-time write-up. Documentation
  for Konfidence MUST be authored and kept up to date in
  [konfidence-project/konfidence-docs](https://github.com/konfidence-project/konfidence-docs),
  not as static, one-off docs left to drift in this repository.

## CI / Approval Policy

- All changes MUST go through a pull request. Direct commits are not a valid path.
- Pushes to `main` are forbidden and prevented by branch protection. Never attempt
  to push directly to `main`, and never suggest bypassing this.
- A PR MUST have at least one approval before merge.
- Automerge MAY be enabled on a PR once required checks and approval are
  satisfied — it is not mandatory per PR.

## Design Boundaries

The section below defines binding architectural boundaries for the design and
implementation of Konfidence, written in the style of an IETF RFC using the
requirement-level keywords from [RFC 2119] / [RFC 8174]: **MUST**, **MUST
NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**,
**RECOMMENDED**, **MAY**, **OPTIONAL**.

This is not a feature backlog. It is a set of scope and ownership boundaries
deliberately drawn between Konfidence and any downstream platform built on
top of it. These boundaries constrain what Konfidence builds, what it
explicitly delegates to integrators, and what guarantees it does or does not
make.

### Applicability — read this before proposing or making a change

1. Every normative statement below (containing MUST, MUST NOT, SHALL, SHALL
   NOT, SHOULD, SHOULD NOT, REQUIRED, RECOMMENDED, MAY, OPTIONAL) is a design
   boundary, not a suggestion.
2. An AI agent MUST treat every MUST/MUST NOT/SHALL/SHALL NOT statement here
   as a hard constraint on any plan, design, or code it produces.
3. If a user request conflicts with a MUST, MUST NOT, SHALL, or SHALL NOT
   clause, the agent MUST NOT silently comply and MUST NOT silently refuse.
   Instead the agent MUST:
   a. Explicitly identify which clause(s) the request conflicts with.
   b. Explain the concrete technical and organizational consequences of
      overriding that boundary (e.g. scope creep into integrator-owned
      territory, loss of a guarantee this document says Konfidence does not
      make, duplicated ownership of a concern, breaking the reference vs.
      production distinction, etc.).
   c. State plainly that changing a boundary here is a **team decision**,
      not something to be settled unilaterally in the course of an
      implementation task.
   d. Explicitly ask the user for confirmation before proceeding, and
      recommend the decision be recorded (e.g. as an ADR) rather than
      implemented silently.
4. Absent an explicit, confirmed override obtained via step 3, agents MUST
   implement work consistently with this document.
5. SHOULD / RECOMMENDED clauses may be deviated from with a clear
   justification in the interaction, but the agent SHOULD still surface the
   deviation rather than silently ignoring the guidance.

### Terminology

- **Vector**: A deployable unit that Konfidence manages through its
  lifecycle (deploy, activate, run, undeploy). A Vector is realized as an
  OCM (Open Component Model) component version; a Stage references it by
  pointing at that component version, and its lifecycle is materialized
  across multiple concrete resources (e.g. the resources that define,
  deploy, activate, assign, promote, and migrate it) rather than a single
  monolithic resource.
- **Stage**: The primary input surface through which a Vector's desired
  state is expressed to Konfidence (see "Stages, Vectors, and Lifecycle").
- **Landscape**: A logical grouping of one or more Deployment Targets that
  Stages and Vectors are deployed into.
- **Deployment Target** (also referred to at the design level as a
  "Runtime"): An independently configurable execution target within a
  Landscape. A Landscape MAY contain multiple Deployment Targets, including
  multiple of the same type. Today's reference deployer resolves at most one
  Deployment Target per Landscape; support for multiple concurrently
  configurable Deployment Targets per Landscape is a forward-looking
  boundary in this document, not yet the deployed reality everywhere.
- **Artifact**: The packaged, versioned output — backed by OCM components
  and resources — that is deployed as part of a Vector, optionally in
  multiple configuration flavors.
- **Orbit**: The lifecycle/management layer around a Landscape's deployment
  targets and deployments, sitting above Konfidence. Konfidence does not
  implement Orbit; it is entirely Integrator-owned (see "Explicit
  Non-Goals").
- **LCP**: Landscape Control Plane — the Konfidence control-plane instance
  responsible for a given set of Landscapes. The LCP is the party on one
  side of the contract with a Landscape's deployer/orchestrator (which may
  run on a different Kubernetes cluster than the LCP itself); the
  LCP-focused UI scopes what it shows to the Landscapes owned by the current
  LCP instance.
- **Deployer**: The component that reconciles Konfidence's deployment-facing
  resources against a Landscape's Deployment Target(s) — e.g. rendering and
  applying artifacts, producing deployment results consumed by later stages
  of the Vector lifecycle (such as routing configuration). Konfidence
  defines the Deployer contract; `kubernetes-landscape-orchestrator` is the
  reference Deployer implementation for Kubernetes-native deployments (see
  "Deployments, Landscapes, and Deployment Targets").
- **Integrator**: Any external product, team, or platform that consumes
  Konfidence as a building block and is responsible for building the
  remaining product surface on top of it (routing, tenancy, orbit
  management, etc.). This document never assumes there is only one such
  Integrator, and Konfidence's public contracts MUST be designed
  Integrator-agnostic.
- **OFREP**: OpenFeature Remote Evaluation Protocol. Konfidence's reference
  deployer exposes an OFREP evaluation endpoint that resolves feature flags,
  authored configuration, and deployer-produced deployment results, keyed by
  Vector ID — the same surface that carries service discovery information
  (see "Service Discovery and Routing Guarantees").

### Ingress and Traffic Routing

1. Konfidence MUST ship a reference implementation of the ingress routing
   path that controls how traffic flows from a user's browser into a
   Vector's context.
2. Konfidence MUST inject the Vector ID and Stage version as headers into
   the HTTP request context as part of this routing path.
3. The reference ingress implementation SHALL be based on a simple, easily
   replaceable reverse proxy ("Easy Proxy") that serves purely as a gateway
   into the Vector context.
4. The reference ingress implementation MUST NOT be presented or relied upon
   as production-grade. It is explicitly insufficient for an Integrator's
   production routing needs.
5. Any Integrator building a production ingress path MUST implement its own
   routing component and header injection mechanism. Konfidence MUST NOT
   assume responsibility for production-grade ingress routing.

### Service Discovery and Routing Guarantees

1. Konfidence MUST offer service discovery information via the OFREP
   protocol's evaluation endpoint, backed by deployer-produced deployment
   results and populated as part of the deployment process.
2. Konfidence SHALL NOT guarantee routing capability or behavior. Routing is
   considered a function of the underlying platform and infrastructure,
   outside Konfidence's contract.
3. Konfidence MAY offer arbitrary, protocol-agnostic discovery information
   as part of its service discovery advertisement mechanism, beyond what
   OFREP itself defines.

### Stages, Vectors, and Lifecycle

1. "Stage" SHALL be the primary input surface for Konfidence. Desired state
   MUST be expressed through Stages rather than through Vector-level or
   Deployment-Target-level primitives directly.
2. Konfidence MUST offer, via its vector data service, visibility into
   Stages and the Vectors assigned to and running under each Stage.
3. Konfidence MUST support task and hook execution during the activation
   phase of a Vector.
4. Konfidence MUST provide a documented mechanism for undeploying Vectors,
   and MUST expose lifecycle hook points into that undeploy process.
5. Konfidence MUST provide a preventative guard mechanism that can block
   decommissioning of a Vector deployment under configured conditions.
6. Konfidence MUST expose a custom property to suspend a Stage from being
   processed.
7. Konfidence MUST make lifecycle state information accessible to an
   external Integrator-owned interface (e.g. a custom controller that reads
   Konfidence resources to determine status), so that Integrators MAY build
   their own controllers against this state rather than Konfidence owning
   that controller itself.
8. Historical data for Vectors and Stages MUST remain accessible after the
   fact (not only current/live state).
9. Usages (e.g. a StageVersionUsage pinning a StageVersion as in-use — a
   resource with a bounded lifetime) SHOULD support a configurable
   expiration after which they are eligible for garbage collection.

### Deployments, Landscapes, and Deployment Targets

1. A single deployment MAY target multiple, independently configurable
   Deployment Targets within one Landscape.
2. Deployment Targets within a Landscape MAY carry different configuration,
   including different Helm values, from one another.
3. A Landscape MAY be composed of multiple Deployment Targets, including
   multiple of the same type.
4. Artifacts MAY ship multiple configuration flavors, applied based on
   matchers configured at the Landscape level.
5. Konfidence MUST support Helm hooks.
6. Konfidence MUST provide a reference/demo implementation of its
   Deployment Interfaces.
7. Konfidence MUST provide a documented way to onboard new Deployment
   Targets.
8. Kubernetes (via a kubernetes-landscape-orchestrator) is the only
   orchestrator Konfidence is REQUIRED to ship. There is no requirement for
   Konfidence to ship support for any other orchestrator.
9. Konfidence MAY offer a non-Vector-based deployment mode (e.g. a
   progressive-delivery style mode comparable to Argo/Kargo) as an
   additional, optional deployment model.

### Explicit Non-Goals: Tenancy and Orbit/Landscape Management

1. Application-level tenant functionality is out of scope for Konfidence.
   Any Integrator MUST build its own tenant functionality on top of
   Konfidence's primitives.
2. Application tenancy is considered an Integrator-side extension.
   Konfidence MUST NOT absorb tenancy concerns into its own scope.
3. Integrators MAY use the Tasks interface to run custom tenant lifecycle
   scripts or migrations; Konfidence MUST support this as a generic
   extension point without encoding tenant-specific logic itself.
4. Konfidence MUST NOT take over landscape or orbit management deployments.
5. Konfidence SHALL NOT own a managed Orbit lifecycle. Any Orbit lifecycle
   management MUST be built by the Integrator on top of Konfidence.

### LCP-Focused UI and Embedding

1. Konfidence MUST offer an LCP-focused UI that surfaces only the
   information relevant to the Landscapes, Stages, and Deployment Targets
   created or managed by the current LCP, including logs, events, and
   runtime information.
2. The UI MUST support embedding of its own UI elements/pages into other
   surfaces (i.e. Konfidence UI components are embeddable elsewhere).
3. The UI SHOULD support being embedded within another web page (i.e.
   Konfidence's UI can itself be hosted inside a third-party page).

### Authentication, Authorization, and Project Management

1. Login MUST work via OIDC.
2. Session data MAY be enriched at login time to inject custom groups.
3. Authorization MUST support a fixed set of roles: `dev`, `admin`, and `pm`.
4. Konfidence MUST offer project management as a mechanism for the logical
   separation of resources.

### Feature Flags and Custom Configuration

1. Konfidence MUST support creating feature flags that are accessible to the
   application running within a Deployment Target.
2. Konfidence MUST support making custom configuration available to
   arbitrary components running within a Deployment Target, including
   components that are not themselves part of a Vector, so that non-Vector
   components can be fed configuration through the same mechanism.

### Observability

1. Controllers MUST expose OTEL-compliant `/metrics` endpoints detailing the
   performance of the underlying deployment lifecycle.

### Rationale Summary (Non-Normative)

The boundaries above consistently draw one line: Konfidence owns Vector,
Stage, Deployment Target, and Landscape primitives, their lifecycle, discovery
metadata, and reference implementations of the interfaces around them.
Konfidence explicitly does not own: production-grade ingress routing,
routing guarantees, application-level tenancy, or Orbit/Landscape lifecycle
management. Those are Integrator responsibilities by design, not omissions.
Any change that moves a capability across this line is a scope change to
the product, not a routine implementation detail — see "Applicability"
above.

[RFC 2119]: https://www.rfc-editor.org/rfc/rfc2119
[RFC 8174]: https://www.rfc-editor.org/rfc/rfc8174
