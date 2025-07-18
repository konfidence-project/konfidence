package setcondition

import (
	"context"
	"github.com/konfidence-project/pkg/conditions"
	"github.com/konfidence-project/pkg/funcopts"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Object = conditions.Setter

const (

	// defaultConditionType to use if non was provided
	defaultConditionType = conditions.ConditionReady

	// defaultReason to use if no reason is provided
	defaultReason = "ResourceAvailable"

	// defaultMessage to use if no message is provided
	defaultMessage = "Resource is now available"
)

// Option is a function that applies options to a Step.
type Option[T Object] func(*Step[T])

// stepOptions holds the options for a Step.
type stepOptions struct {
	conditionType conditions.ConditionType
	reason        string
	message       string
}

// WithConditionType allows to specify a custom condition type
func WithConditionType[T Object](t conditions.ConditionType) Option[T] {
	return func(step *Step[T]) {
		step.options.conditionType = t
	}
}

// WithReason sets the reason for the step.
func WithReason[T Object](s string) Option[T] {
	return func(step *Step[T]) {
		step.options.reason = s
	}
}

// WithMessage sets the message for the step.
func WithMessage[T Object](s string) Option[T] {
	return func(step *Step[T]) {
		step.options.message = s
	}
}

// Step is a struct that implements the pipeline.Step interface.
type Step[T Object] struct {
	options stepOptions
	status  v1.ConditionStatus
}

// New creates a new Step with the provided status and options.
func New[T Object](status v1.ConditionStatus, options ...Option[T]) *Step[T] {
	s := &Step[T]{
		options: stepOptions{
			conditionType: defaultConditionType,
			reason:        defaultReason,
			message:       defaultMessage,
		},
		status: status,
	}

	return funcopts.Apply[*Step[T], Option[T]](s, options)
}

// Run implements the pipeline.Step interface for Step.
func (s Step[T]) Run(ctx context.Context, c client.Client, obj T) (ctrl.Result, error) {

	conditions.Set(obj, conditions.NewCondition(
		s.options.conditionType,
		s.status,
		s.options.reason,
		s.options.message,
	))

	if err := c.Status().Update(ctx, obj); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
