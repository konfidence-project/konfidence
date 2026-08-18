package ocm

import (
	ocmcredentials "ocm.software/open-component-model/bindings/go/credentials"

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

// AdapterOption is a functional option for configuring the Adapter.
type AdapterOption func(*Adapter)

// WithOCMClient sets the underlying OCI/OCM repository client on the Adapter.
func WithOCMClient(c pkgocm.Client) AdapterOption {
	return func(a *Adapter) {
		a.ocmClient = c
	}
}

// WithVectorSigner sets a Signer used to sign vectors. If no signer is provided vector signing is disabled.
func WithVectorSigner(signer crypto.Signer) AdapterOption {
	return func(a *Adapter) {
		a.vectorSigner = signer
	}
}

// WithVerifier sets the shared Verifier used to verify vectors and artifacts.
// The verifier is stateless with respect to specs and credentials; per-CR
// specs and resolver are supplied to Verify calls via the fields configured
// through WithResolver, WithVectorVerifySpecs, and WithArtifactVerifySpecs.
func WithVerifier(v crypto.Verifier) AdapterOption {
	return func(a *Adapter) {
		a.verifier = v
	}
}

func WithResolver(r ocmcredentials.Resolver) AdapterOption {
	return func(a *Adapter) {
		a.resolver = r
	}
}

// WithVectorVerifySpecs sets the SignatureSpecs used when verifying vector descriptors.
// Empty slice disables vector verification.
func WithVectorVerifySpecs(specs []crypto.SignatureSpec) AdapterOption {
	return func(a *Adapter) {
		a.vectorSpecs = specs
	}
}

// WithArtifactVerifySpecs sets the SignatureSpecs used when verifying artifact descriptors.
// Empty slice disables artifact verification.
func WithArtifactVerifySpecs(specs []crypto.SignatureSpec) AdapterOption {
	return func(a *Adapter) {
		a.artifactSpecs = specs
	}
}
