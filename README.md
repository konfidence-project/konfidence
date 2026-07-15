[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/konfidence)](https://api.reuse.software/info/github.com/konfidence-project/konfidence)

# konfidence

## About this project

konfidence

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

For API-backed login, start Dex and the API in separate terminals, then start
the dashboard:

```sh
make idp-up
make run-api-with-idp
KONFIDENCE_API_URL=http://localhost:8090 pnpm ui:dev
```

Open `http://localhost:5173/landscape` and sign in with
`alice@example.com` / `password`. The dashboard proxies `/api/*` to the API,
which owns the OAuth flow and session.

The dashboard is packaged as `konfidence-ui` and is built into the default Docker Bake target together with the star and galaxy operator images. In CI, the dashboard check job runs formatting, linting, type checks, and Svelte checks; the production build runs through the dashboard image build.

Deploy the star chart with the dashboard enabled:

```sh
helm upgrade --install star charts/star \
  --set dashboard.enabled=true \
  --set image.tag=<tag> \
  --set dashboard.image.tag=<tag>
```

For local Makefile deployment, `make deploy-star REGISTRY=<registry> TAG=<tag> DASHBOARD_ENABLED=true` passes both the star operator and dashboard image repository/tag values into the chart and enables the dashboard.

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/konfidence-project/<your-project>/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/konfidence-project/<your-project>/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2026 SAP SE or an SAP affiliate company and konfidence contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/konfidence-project/<your-project>).
