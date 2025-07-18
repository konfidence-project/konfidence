[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/pkg)](https://api.reuse.software/info/github.com/konfidence-project/pkg)

# pkg

## About this project

The pkg repository contains commonly used libraries for Kubernetes controllers. It is designed to simplify the development of Kubernetes operators by providing reusable components for managing conditions, pipelines, and steps.

## Features

- **Conditions**: Manage and evaluate conditions for Kubernetes resources.
- **Pipelines**: Create and execute pipelines for resource reconciliation.
- **Steps**: Implement reusable steps for pipeline execution.

## Installation

To use the libraries in your project, add them as a dependency:

```bash
go get github.com/konfidence-project/pkg
```

## Usage

### Pipeline examples

#### Basic Pipeline

```go
package stagecontroller

import (
	"context"
	"github.com/konfidence-project/pkg/conditions"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"fmt"
	"github.com/konfidence-project/pkg/pipeline"
	"github.com/konfidence-project/pkg/steps/ensurefinalizer"
	"github.com/konfidence-project/pkg/steps/setcondition"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	MyCustomCondition conditions.ConditionType = "MyCustomCondition"
)

// Usual reconciler loop for a Kubernetes controller

func (r *StageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var resource v1alpha1.Stage

	// Create a pipeline for the Stage resource
	p, err := pipeline.New(&resource)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Add pipeline steps to handle the Stage resource lifecycle
	p.AddStep(

		// Set a ready condition to false at the beginning of the reconciliation
		setcondition.New[*v1alpha1.Stage](
			v1.ConditionFalse,
			setcondition.WithReason("ReconcileInProgress"),
			setcondition.WithMessage("Reconciliation is in progress"),
		),

		// Ensure finalizer is present before proceeding with deletion or creation
		// This is a tested reusable step!
		ensurefinalizer.New[*v1alpha1.Stage](
			common.FinalizerName,
		),

		// Add a step as a function to keep things simple and flexible
		pipeline.StepFunc[*v1alpha1.Stage](func(ctx context.Context, c k8sclient.Client, obj *v1alpha1.Stage) (ctrl.Result, error) {

			// Just return an error if something is wrong
			if somethingIsWrong {
				return ctrl.Result{}, fmt.Errorf("something went wrong")
			}

			// Stop the reconciliation if a certain condition is met
			// and return an error to avoid further processing
			if shouldStopReconcile {
				return ctrl.Result{}, pipeline.ErrPipelineBreak
			}

			// Requeue the reconciliation if a condition is met
			// This will also stop the pipeline execution
			if shouldBeRequed {
				return ctrl.Result{RequeueAfter: time.Second * 5}, nil
			}

			// Return to run next step in the pipeline
			return ctrl.Result{}, nil
		}),

		// Add your reusable step here
		NewMyReusableStep[*v1alpha1.Stage](),

		// Set a ready condition on the resource
		setcondition.New[*v1alpha1.Stage](),

		// Set a custom condition on the resource
		setcondition.New[*v1alpha1.Stage](
			v1.ConditionTrue,
			setcondition.WithConditionType(MyCustomCondition),
			setcondition.WithReason("ReconcileInProgress"),
			setcondition.WithMessage("Reconciliation is in progress"),
		),
	)

	// Run the pipeline with the provided context, client, and request
	return p.Run(ctx, r.Client, req.NamespacedName)
}

// Example of a reusable step
type MyReusableStep[T pipeline.Object] struct {
	// Define any fields needed for the step
}

func NewMyReusableStep[T pipeline.Object]() *MyReusableStep[T] {
	return &MyReusableStep{
		// Initialize fields if necessary
	}
}

// Implement the Step interface for your reusable step
func (s *MyReusableStep[T]) Run(ctx context.Context, c k8sclient.Client, obj T) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

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
