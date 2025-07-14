package ensurefinalizer

import (
	"context"

	pipeline "github.tools.sap/konfidence/pkg/pipeline"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type Step[T pipeline.Object] struct {
	finalizerName string
}

func New[T pipeline.Object](finalizerName string) *Step[T] {
	return &Step[T]{
		finalizerName: finalizerName,
	}
}

func (s Step[T]) Run(ctx context.Context, c client.Client, obj T) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("finalizer", s.finalizerName)

	// Check if the object is not being deleted and does not have the finalizer
	if obj.GetDeletionTimestamp().IsZero() && !controllerutil.ContainsFinalizer(obj, s.finalizerName) {
		log.Info("Adding finalizer to resource")

		controllerutil.AddFinalizer(obj, s.finalizerName)
		err := c.Update(ctx, obj)
		if err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}
