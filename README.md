[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/konfidence)](https://api.reuse.software/info/github.com/konfidence-project/konfidence)

# Konfidence

## Description

Konfidence is a cloud-native deployment orchestration platform that enables safe and traceable delivery of multi-component applications to your target platforms. 
Modern applications often consist of multiple interconnected services that need to be deployed, tested and published as cohesive units while managing complex dependencies and configurations.
Konfidence addresses these challenges by treating application deployments as immutable vectors that can be safely deployed across multiple environments.

At its core, Konfidence solves the problem of coordinating multi-service deployments with confidence. 
Traditional deployment tools focus on individual components, but Konfidence allows you to define complete application configurations as vectors—versioned, immutable deployment units that reference all required artifacts, configuration and orchestration tasks. 
This approach enables teams to deploy the same application configuration consistently across development, staging and production environments.

This repository contains the core Konfidence operators, the Konfidence API server and the Konfidence CLI. 
For comprehensive documentation, architecture details, and getting started guides, visit [konfidence.cloud](https://konfidence.cloud).

> **Warning**: This project is in early development. The APIs and features are subject to change.
> We do not give any guarantees regarding compatibility or stability at this stage. 
> Please reach out to us if you are interested in early adoption or collaboration.

## Installation

Konfidence runs on a Kubernetes cluster and consists of two main components:

- **Konfidence operator**: Orchestrates delivery flows across stages and manages vector lifecycles
- **Kubernetes landscape orchestrator**: Reference deployer implementation for Kubernetes deployments

You can quickly set up a local test environment using kind or install Konfidence on an existing Kubernetes cluster using Helm charts.

For a step-by-step guide including cluster setup, component installation, and your first vector deployment, see the [Quickstart Guide](https://konfidence.cloud/docs/getting-started/quickstart.html).

For detailed installation instructions and production deployment considerations, see the [Installation Guide](https://konfidence.cloud/docs/deploy-operate/installation.html).

## Local Development

Beyond the kind and Helm setup described in Installation, the repository has make targets for
running the operator and API server directly on your host against local dependencies. Activate
Hermit and start the local dependencies (an IDP via Authelia, a reverse proxy via Caddy, and
Postgres) before running anything else:

```sh
source ./bin/activate-hermit
make dev-up
```

Use `make dev-down` to stop these dependencies and `make dev-logs` to tail their logs.

A local kind cluster with a local OCI registry is available for installing the operator and
pushing vectors and artifacts without a real registry:

```sh
make dev-cluster
```

This creates a kind cluster named `konfidence-dev` and a registry at `localhost:5001`. Build and
push images against it with `make docker-build docker-push` for the operator and `make
docker-build-api docker-push-api` for the API server, both pointed at the registry with
`REGISTRY=localhost:5001`. These targets cross-compile a Linux binary regardless of your host
OS, since the images are Linux-based, so they work the same way on macOS and Linux. With images
pushed, deploy the operator on its own with:

```sh
REGISTRY=localhost:5001 make docker-build docker-push
make deploy
```

Deploying the full stack, including the API server, additionally needs the local Authelia
instance (started by `dev-up`) wired in as its OIDC provider, and its local CA trusted by the
pod so it can reach Authelia over HTTPS:

```sh
REGISTRY=localhost:5001 TAG=dev \
DEPLOY_OIDC_ISSUER_URL=https://host.docker.internal \
DEPLOY_OIDC_CLIENT_ID=konfidence \
DEPLOY_OIDC_REDIRECT_URL=https://api.localhost/api/v1/auth/callback \
DEPLOY_OIDC_CLIENT_SECRET=konfidence-local-secret \
DEPLOY_OIDC_ALLOW_RETURN_URLS=https://api.localhost \
DEPLOY_OIDC_TRUST_CADDY_CA=1 \
make deploy
```

`host.docker.internal` is how pods inside the cluster reach the Authelia instance running on the
host; `DEPLOY_OIDC_TRUST_CADDY_CA` pulls Caddy's local CA certificate out of its container and
mounts it into the API pod so that connection is trusted.

With the dependencies and cluster running, generate webhook certificates once with `make
webhook-certs`, then run the operator with `make run` and the API server with `make
run-kden-api`. The API server's OIDC flags already point at the Authelia instance started by
`dev-up`, so no further configuration is needed there.

The `kden` CLI's `api-endpoint` also defaults to `http://localhost:8090`, matching
`run-kden-api`, so no configuration is needed to talk to a locally running API server. Pushing
vectors and artifacts to the local registry needs an explicit scheme in `--registry`, since the
OCM client defaults to HTTPS and fails against the registry container's plain HTTP:

```sh
kden vector push --file vector.yaml --registry=http://localhost:5001/<subpath>
```

## Dashboard Development

The production dashboard lives in `apps/konfidence-ui`. Activate Hermit and install the workspace dependencies before starting it:

```sh
source ./bin/activate-hermit
pnpm install
pnpm ui:dev
```

The development server proxies `/api/v1` requests to the API at `http://127.0.0.1:8090`. Run the API separately with `make run-kden-api` when developing API-backed features.

Install Chromium once before running browser-based tests:

```sh
pnpm --filter konfidence-ui exec playwright install chromium
```

Run the static checks and each test layer with repository commands:

```sh
pnpm ui:check
pnpm ui:check:svelte
pnpm ui:lint
pnpm ui:fmt:check
pnpm ui:test:unit
pnpm ui:test:browser
pnpm ui:test:e2e
```

`pnpm ui:verify` runs all static checks, while `pnpm ui:test` runs every test layer. `pnpm ui:all` runs both groups.

Build and preview the static SPA with:

```sh
pnpm ui:build
pnpm ui:start
```

The preview server is intended for checking the root page. To exercise production SPA fallback behavior, serve the build through the API with `go run ./cmd/api --ui-asset-path=apps/konfidence-ui/build`.

Production releases include the static SPA in the API image. The API serves the dashboard and `/api/v1` from the same origin; set `API_UI_ASSET_PATH` when running an API binary with an external dashboard build directory.

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/konfidence-project/konfidence/issues). 
Contribution and feedback are encouraged and always welcome.
For more information about how to contribute see our [Contribution Guidelines](https://github.com/konfidence-project/.github/blob/main/CONTRIBUTING.md).

## Security / Disclosure

If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/konfidence-project/.github/blob/main/SECURITY.md) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2026 SAP SE or an SAP affiliate company and konfidence-project contributors. 
Please see our [LICENSES](LICENSES) for copyright and license information. 
Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/konfidence-project/konfidence).
