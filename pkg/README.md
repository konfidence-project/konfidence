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

### Pipeline examples

#### Basic Pipeline

```go
package stagecontroller

import (
	"context"
	"time"

	"fmt"
	"github.tools.sap/konfidence/pkg/pipeline"
	"github.tools.sap/konfidence/pkg/steps/ensurefinalizer"
	ctrl "sigs.k8s.io/controller-runtime"
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
