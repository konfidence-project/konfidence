package pipeline

import (
	"context"
	"errors"
	"reflect"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrPipelineBreak is an error that can be returned by a step function to indicate
// that the pipeline should stop processing further steps.
var ErrPipelineBreak = errors.New("pipeline break requested")

type (

	// Step is an interface that defines a step in the pipeline.
	Step[T Object] interface {
		Run(ctx context.Context, c client.Client, obj T) (ctrl.Result, error)
	}

	// StepFunc is a function type that represents a step in the pipeline.
	StepFunc[T Object] func(ctx context.Context, c client.Client, obj T) (ctrl.Result, error)
)

// Run implements Step interface for StepFunc.
func (f StepFunc[T]) Run(ctx context.Context, c client.Client, obj T) (ctrl.Result, error) {
	return f(ctx, c, obj)
}

// Pipeline is a generic type that represents a sequence of steps to be executed on a Kubernetes object.
// It is parameterized by a type T that implements the Object interface.
type Pipeline[T Object] struct {
	obj   T
	steps []Step[T]
}

// New creates a new Pipeline instance for the specified type T.
func New[T Object](obj T) (*Pipeline[T], error) {

	// Object must not be nil, as it is required for the pipeline to function.
	if reflect.TypeOf(obj).Kind() != reflect.Ptr {
		return nil, errors.New("pipeline object is not a pointer, expected a pointer to an Object type")
	}

	return &Pipeline[T]{
		obj:   obj,
		steps: make([]Step[T], 0),
	}, nil
}

// AddStep adds a step function to the pipeline. Each step function takes a context, a client, and an object of type T.
func (p *Pipeline[T]) AddStep(step ...Step[T]) *Pipeline[T] {
	p.steps = append(p.steps, step...)
	return p
}

// AddStepFunc adds a step function to the pipeline using a variadic parameter.
func (p *Pipeline[T]) AddStepFunc(stepFuncs ...StepFunc[T]) *Pipeline[T] {

	// Convert the slice of StepFunc to a slice of Step[T].
	steps := make([]Step[T], len(stepFuncs))
	for i, stepFunc := range stepFuncs {
		steps[i] = stepFunc
	}

	return p.AddStep(steps...)
}

// GetObject returns the Kubernetes object associated with the pipeline.
func (p *Pipeline[T]) GetObject() T {
	return p.obj
}

// GetSteps returns the list of steps in the pipeline.
func (p *Pipeline[T]) GetSteps() []Step[T] {
	return p.steps
}

// Run executes the pipeline steps on the specified Kubernetes object identified by the provided key.
func (p *Pipeline[T]) Run(ctx context.Context, c client.Client, key client.ObjectKey) (ctrl.Result, error) {

	// Step out if count of steps is zero, as there is nothing to execute.
	if len(p.steps) == 0 {
		return ctrl.Result{}, nil
	}

	// Retrieve the object from the Kubernetes API using the provided key.
	if err := c.Get(ctx, key, p.obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Iterate through each step in the pipeline and execute it.
	for _, step := range p.steps {
		res, err := step.Run(ctx, c, p.obj)

		// If the step requests a requeue after a specific duration, we return the result.
		if res.Requeue || res.RequeueAfter > 0 {
			return res, nil
		}

		// If the step returns ErrPipelineBreak, we stop processing further steps.
		if errors.Is(err, ErrPipelineBreak) {
			return res, nil
		}

		// If an error occurs or the step requests a requeue, we return the result and error.
		if err != nil {
			return ctrl.Result{}, err // We do not return the result here, as it may contain a requeue request.
		}
	}

	return ctrl.Result{}, nil
}
