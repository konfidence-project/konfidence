package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	vectorocm "github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/ocm"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/vector"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	"github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

// NewCacheFactory returns the production clientcache factory for VectorAssembly.
// Exported so that setup.go and envtest suites both use the exact same production
// code path — no duplication, no drift.
func NewCacheFactory(log logr.Logger, limiter crypto.Limiter) clientcache.Factory[*v1alpha1.VectorTemplate, vector.OcmPort] {
	return func(ctx context.Context, k8sClient client.Reader, cr *v1alpha1.VectorTemplate) (vector.OcmPort, error) {
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
		artifactVerifier, err := crypto.NewVerifierBuilder().
			WithSpecs(crypto.SpecsFromVerify(cr.Spec.VerifyArtifacts)).
			WithResolver(resolver).
			WithLimiter(limiter).
			Build()
		if err != nil {
			return nil, fmt.Errorf("building artifact verifier: %w", err)
		}
		signer, err := crypto.NewSignerBuilder().
			WithSpecs(crypto.SpecsFromSign(cr.Spec.SignVector)).
			WithResolver(resolver).
			WithLimiter(limiter).
			Build()
		if err != nil {
			return nil, fmt.Errorf("building signer: %w", err)
		}
		return vectorocm.NewAdapter(
			vectorocm.WithOCMClient(ociClient),
			vectorocm.WithVectorVerifier(vectorVerifier),
			vectorocm.WithArtifactVerifier(artifactVerifier),
			vectorocm.WithVectorSigner(signer),
		), nil
	}
}
