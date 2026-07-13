package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/ocm"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	"github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

// NewCacheFactory returns the production clientcache factory for VectorPromotion.
// Exported so that setup.go and envtest suites both use the exact same production
// code path — no duplication, no drift.
func NewCacheFactory(log logr.Logger, limiter crypto.Limiter) clientcache.Factory[*konfidence.VectorPromotionConfig, promotion.OcmPort] {
	return func(ctx context.Context, k8sClient client.Reader, cr *konfidence.VectorPromotionConfig) (promotion.OcmPort, error) {
		resolver, err := credentials.ResolverFromCredentials(ctx, k8sClient, cr.Namespace, cr.Spec.Credentials)
		if err != nil {
			return nil, fmt.Errorf("resolving credentials: %w", err)
		}
		ociClient, err := repository.NewOciClientBuilder().WithResolver(resolver).WithLogger(log).Build(ctx)
		if err != nil {
			return nil, fmt.Errorf("building OCI client: %w", err)
		}
		vectorVerifier, err := crypto.NewVerifierBuilder().
			WithSpecs(crypto.SpecsFromVerify(cr.Spec.VerifyVector)).
			WithResolver(resolver).
			WithLimiter(limiter).
			Build()
		if err != nil {
			return nil, fmt.Errorf("building vector verifier: %w", err)
		}
		return ocm.NewPromotionAdapter(
			ocm.WithOCMClient(ociClient),
			ocm.WithVectorVerifier(vectorVerifier),
		), nil
	}
}
