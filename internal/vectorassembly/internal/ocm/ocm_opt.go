package ocm

import (
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

// WithArtifactVerifier sets a Verifier to verify artifacts. If no verifier is provided artifact verification is disabled.
func WithArtifactVerifier(verifier crypto.Verifier) AdapterOption {
	return func(a *Adapter) {
		a.artifactVerifier = verifier
	}
}

// WithVectorVerifier sets a Verifier to verify vectors. If no verifier is provided vector verification is disabled.
func WithVectorVerifier(verifier crypto.Verifier) AdapterOption {
	return func(a *Adapter) {
		a.vectorVerifier = verifier
	}
}
