package funcopts

import (
	"errors"
)

// Apply can be used to apply functional options to an object instance.
// T is the type of instance that the options should be applied on
// O is the type of options function that is used
// See tests for examples
func Apply[T any, O ~func(T)](instance T, options []O) T {
	for _, option := range options {
		option(instance)
	}

	return instance
}

// ApplyWithErrors is same as Apply, but expects an options function that returns an error
func ApplyWithErrors[T any, O ~func(T) error](instance T, options []O) (T, error) {
	var (
		zero T
		errs []error
	)
	for _, option := range options {
		err := option(instance)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return zero, errors.Join(errs...)
	}

	return instance, nil
}
