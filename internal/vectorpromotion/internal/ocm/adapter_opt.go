package ocm

import (
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgrepository "github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

// PromotionAdapterOption is a functional option for configuring the PromotionAdapter.
type PromotionAdapterOption func(*PromotionAdapter)

// WithVectorVerifier sets a Verifier to verify vectors. If no verifier is provided vector verification is disabled.
func WithVectorVerifier(verifier crypto.Verifier) PromotionAdapterOption {
	return func(a *PromotionAdapter) {
		a.vectorVerifier = verifier
	}
}

// WithOCMClient sets the underlying OCI/OCM repository client on the PromotionAdapter.
func WithOCMClient(c pkgrepository.Client) PromotionAdapterOption {
	return func(a *PromotionAdapter) {
		a.ocmClient = c
	}
}
