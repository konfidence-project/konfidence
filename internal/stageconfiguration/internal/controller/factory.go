package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/ocm"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/ports"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	"github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

// NewCacheFactory returns the production clientcache factory for StageConfiguration.
// Exported so that setup.go and envtest suites both use the exact same production
// code path — no duplication, no drift.
func NewCacheFactory(log logr.Logger, limiter crypto.Limiter) clientcache.Factory[*konfidence.StageConfiguration, ports.VectorPort] {
	return func(ctx context.Context, k8sClient client.Reader, cr *konfidence.StageConfiguration) (ports.VectorPort, error) {
		resolver, err := credentials.ResolverFromCredentials(ctx, k8sClient, cr.Namespace, cr.Spec.Credentials)
		if err != nil {
			return ocm.VectorOCMAdapter{}, fmt.Errorf("resolving credentials: %w", err)
		}
		ociClient, err := repository.NewOciClientBuilder().WithResolver(resolver).WithLogger(log).Build(ctx)
		if err != nil {
			return ocm.VectorOCMAdapter{}, fmt.Errorf("building OCI client: %w", err)
		}
		vectorVerifier, err := crypto.NewVerifierBuilder().
			WithSpecs(crypto.SpecsFromVerify(cr.Spec.VerifyVector)).
			WithResolver(resolver).
			WithLimiter(limiter).
			Build()
		if err != nil {
			return ocm.VectorOCMAdapter{}, fmt.Errorf("building vector verifier: %w", err)
		}
		return ocm.VectorOCMAdapter{VectorVerifier: vectorVerifier, OcmClient: ociClient}, nil
	}
}
