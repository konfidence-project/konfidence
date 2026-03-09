package ocm

import (
	"context"
	"fmt"

	"github.com/konfidence-project/pkg/ocm/crypto"
	pkgOcm "github.com/konfidence-project/pkg/ocm/repository"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// AdapterOption is a functional option for configuring the Adapter.
type AdapterOption func(*Adapter)

// WithVectorSigner sets a Signer used to sign vectors. If no signer is provided vector signing is disabled.
// This is useful for testing purposes, e.g. to inject a mock Signer that does not perform actual signing.
func WithVectorSigner(signer crypto.Signer) AdapterOption {
	return func(a *Adapter) {
		a.vectorSigner = signer
	}
}

// WithDigester sets a Digester instead of using the default digest generation implementation.
// This is useful for testing purposes, e.g. to inject a mock Digester that does not perform actual digest generation.
func WithDigester(digester crypto.Digester) AdapterOption {
	return func(a *Adapter) {
		a.digester = digester
	}
}

// WithArtifactVerifier sets a Verifier to verify artifacts. If no verifier is provided artifact verification is disabled.
// This is useful for testing purposes, e.g. to inject a mock Verifier that does not perform actual verification.
func WithArtifactVerifier(verifier crypto.Verifier) AdapterOption {
	return func(a *Adapter) {
		a.artifactVerifier = verifier
	}
}

// WithVectorVerifier sets a Verifier to verify vectors. If no verifier is provided vector verification is disabled.
// This is useful for testing purposes, e.g. to inject a mock Verifier that does not perform actual verification.
func WithVectorVerifier(verifier crypto.Verifier) AdapterOption {
	return func(a *Adapter) {
		a.vectorVerifier = verifier
	}
}

// WithDefaultArtifactVerification enables verification of artifacts.
// It will use crypto.ArtifactSignature as the target signature name for verification
// and the given crypto.ConfigMapTrustAnchorProvider as trust anchor config.
// If setup fails this option will panic.
func WithDefaultArtifactVerification(provider *crypto.ConfigMapTrustAnchorProvider) AdapterOption {
	return func(a *Adapter) {
		rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.ArtifactSignature},
			crypto.WithCredentialProvider(provider))
		if err != nil {
			panic(fmt.Sprintf("unable to set up default ocm artifact verification: %v", err))
		}

		a.artifactVerifier = rsaVerifier
	}
}

// WithDefaultVectorVerification enables verification of vectors.
// It will use crypto.VectorAssemblySignature as the target signature name for verification
// and the given crypto.ConfigMapTrustAnchorProvider as trust anchor config.
// If setup fails this option will panic.
func WithDefaultVectorVerification(provider *crypto.ConfigMapTrustAnchorProvider) AdapterOption {
	return func(a *Adapter) {
		rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.VectorAssemblySignature},
			crypto.WithCredentialProvider(provider))
		if err != nil {
			panic(fmt.Sprintf("unable to set up default ocm vector verification: %v", err))
		}

		a.vectorVerifier = rsaVerifier
	}
}

// WithDefaultVectorSigning enables default signing of vectors.
// It will use crypto.VectorAssemblySignature as the target signature name for signing
// and the given crypto.SecretSigningCredentialsProvider as credentials provider config.
// If setup fails this option will panic.
func WithDefaultVectorSigning(provider *crypto.SecretSigningCredentialsProvider) AdapterOption {
	return func(a *Adapter) {
		s, err := crypto.NewRSASigner(provider, []string{crypto.VectorAssemblySignature})
		if err != nil {
			panic(fmt.Sprintf("unable to set up default ocm vector signing: %v", err))
		}
		a.vectorSigner = s
	}
}

// WithOcmClient sets the OCM client used to interact with OCM repositories.
// It will use the provided k8s Secret to access OCI registries.
// If the secret is nil, then the OCM client will be built without credentials and use a noop credential resolver.
// If setup fails this option will panic.
func WithOcmClient(ctx context.Context, secret *v1.Secret) AdapterOption {
	return func(a *Adapter) {
		ocmClient, err := pkgOcm.NewOciClientBuilder().
			WithLogger(ctrl.Log).
			WithDockerConfigJsonSecret(secret).
			Build(ctx)
		if err != nil {
			panic(fmt.Sprintf("unable to create ocm client: %v", err))
		}
		a.ocmClient = ocmClient
	}
}
