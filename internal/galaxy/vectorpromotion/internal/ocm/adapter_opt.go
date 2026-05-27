package ocm

import (
	"fmt"

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
)

// PromotionAdapterOption is a functional option for configuring the PromotionAdapter.
type PromotionAdapterOption func(*PromotionAdapter)

// WithVectorVerifier sets a Verifier to verify vectors. If no verifier is provided vector verification is disabled.
func WithVectorVerifier(verifier crypto.Verifier) PromotionAdapterOption {
	return func(a *PromotionAdapter) {
		a.vectorVerifier = verifier
	}
}

// WithDefaultVectorVerification enables verification of vectors.
// It will use crypto.VectorAssemblySignature as the target signature name for verification
// and the given crypto.ConfigMapTrustAnchorProvider as trust anchor config.
// If setup fails this option will panic.
func WithDefaultVectorVerification(provider *crypto.ConfigMapTrustAnchorProvider) PromotionAdapterOption {
	return func(a *PromotionAdapter) {
		rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.VectorAssemblySignature},
			crypto.WithCredentialProvider(provider))
		if err != nil {
			panic(fmt.Sprintf("unable to set up default ocm vector verification: %v", err))
		}

		a.vectorVerifier = rsaVerifier
	}
}
