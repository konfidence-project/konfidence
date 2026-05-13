package vectorpromotion

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/galaxy/vector-promotion/internal/controller"
	"github.com/konfidence-project/konfidence/internal/galaxy/vector-promotion/internal/controller/ocm"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"k8s.io/apimachinery/pkg/runtime"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

// Options configures the vector promotion controllers.
type Options struct {
	// VectorVerificationProvider is an optional ConfigMapTrustAnchorProvider for vector verification.
	// If nil, vector verification is disabled.
	VectorVerificationProvider *crypto.ConfigMapTrustAnchorProvider
}

// SetupControllers registers all vector promotion controllers with the given manager.
func SetupControllers(ctx context.Context, mgr mcmanager.Manager, scheme *runtime.Scheme, opts Options) error {
	var promotionAdapterConfig []ocm.PromotionAdapterOption
	if opts.VectorVerificationProvider != nil {
		promotionAdapterConfig = append(promotionAdapterConfig, ocm.WithDefaultVectorVerification(opts.VectorVerificationProvider))
	}

	if err := (&controller.VectorPromotionReconciler{
		Mgr:               mgr,
		Scheme:            scheme,
		PortProvider:      ocm.NewPromotionPortProvider(promotionAdapterConfig...),
		OcmClientProvider: repository.DefaultOciClientProvider,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controller.VectorPromotionTTLReconciler{
		Mgr:    mgr,
		Scheme: scheme,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controller.VectorPromotionStatusPropagationReconciler{
		Mgr:    mgr,
		Scheme: scheme,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
