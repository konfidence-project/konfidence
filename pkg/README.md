[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/pkg)](https://api.reuse.software/info/github.com/konfidence-project/pkg)

# pkg

## About this project

The pkg repository contains commonly used libraries for Kubernetes controllers. It is designed to simplify the development of Kubernetes operators by providing reusable components for managing conditions, pipelines, and steps.

## Features

- **Conditions**: Manage and evaluate conditions for Kubernetes resources.

## Installation

To use the libraries in your project, add them as a dependency:

```bash
go get github.com/konfidence-project/pkg
```

## Development

### Prerequisites

- Go 1.20+
- Kubernetes client-go

### Running Tests

This project uses Ginkgo and Gomega for testing. Run the tests with:

```bash
make test
```

### Generating Mocks

Mocks are generated using `mockgen`. To regenerate mocks:

```bash
make generate
```

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/konfidence-project/<your-project>/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/konfidence-project/<your-project>/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and pkg contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/konfidence-project/<your-project>).
