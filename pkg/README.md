# pkg

The `pkg` repository contains commonly used libraries for Kubernetes controllers. It is designed to simplify the development of Kubernetes operators by providing reusable components for managing conditions, pipelines, and steps.

## Features

- **Conditions**: Manage and evaluate conditions for Kubernetes resources.
- **Pipelines**: Create and execute pipelines for resource reconciliation.
- **Steps**: Implement reusable steps for pipeline execution.

## Installation

To use the libraries in your project, add them as a dependency:

```bash
go get github.tools.sap/konfidence/pkg
```

## Usage

TBD ... 

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
