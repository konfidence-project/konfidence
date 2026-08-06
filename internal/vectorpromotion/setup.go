package vectorpromotion

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/controller"
)

const OperatorFlagName = "VectorPromotion"

// Options configures the vector promotion controllers. It is empty while
// promotion execution is stubbed pending the ADR-0032 execution rework.
type Options struct{}

// SetupControllers registers all vector promotion controllers with the given manager.
func SetupControllers(_ context.Context, mgr ctrl.Manager, _ Options) error {
	if err := controller.NewVectorPromotionReconciler(mgr).
		SetupWithManager(mgr); err != nil {
		return err
	}

	return controller.NewVectorPromotionTTLReconciler(mgr).
		SetupWithManager(mgr)
}
