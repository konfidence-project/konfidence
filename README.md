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

For a step-by-step guide including cluster setup, component installation, and your first vector deployment, see the [Quickstart Guide](https://preview.konfidence.cloud/docs/getting-started/quickstart.html).

For detailed installation instructions and production deployment considerations, see the [Installation Guide](https://konfidence.cloud/docs/deploy-operate/installation.html).

## Requirements and Setup

Use Hermit for the repository toolchain:

```sh
source ./bin/activate-hermit
pnpm install
```

Common development commands:

```sh
make verify      # Go checks/tests plus dashboard fmt/lint/typecheck
make build       # Build operators and dashboard assets
make docker-bake # Build all container images with Docker Buildx Bake
```

Dashboard-focused commands:

```sh
make verify-ui       # Run dashboard fmt check, lint, typecheck, and Svelte checks
pnpm ui:dev          # Start the SvelteKit development server
pnpm ui:fmt          # Format dashboard sources
pnpm ui:test         # Run dashboard unit/browser-mode and e2e tests
```

The dashboard is packaged as `konfidence-ui` and is built into the default Docker Bake target together with the Konfidence operator image. In CI, the dashboard check job runs formatting, linting, type checks, and Svelte checks; the production build runs through the dashboard image build.

Deploy the Konfidence chart with the dashboard enabled:

```sh
helm upgrade --install konfidence charts/konfidence \
  --set dashboard.enabled=true \
  --set image.tag=<tag> \
  --set dashboard.image.tag=<tag>
```

For local Makefile deployment, `make deploy REGISTRY=<registry> TAG=<tag> DASHBOARD_ENABLED=true` passes both the operator and dashboard image repository/tag values into the chart and enables the dashboard.

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
