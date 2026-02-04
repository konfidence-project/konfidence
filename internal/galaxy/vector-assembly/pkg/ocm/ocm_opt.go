package ocm

import (
	"fmt"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
)

// For now, we assume a single trust anchor for both artifact and vector verification
const (
	VerifierTrustAnchorConfigMapNameEnv      = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME"
	VerifierTrustAnchorConfigMapNamespaceEnv = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE"
)

// AdapterOption is a functional option for configuring the Adapter.
type AdapterOption func(*Adapter)

// WithVectorSigner sets a Signer instead of using the default signing implementation.
// This is useful for testing purposes, e.g. to inject a mock Signer that does not perform actual signing.
func WithVectorSigner(signer Signer) AdapterOption {
	return func(a *Adapter) {
		a.vectorSigner = signer
	}
}

// WithDigester sets a Digester instead of using the default digest generation implementation.
// This is useful for testing purposes, e.g. to inject a mock Digester that does not perform actual digest generation.
func WithDigester(digester Digester) AdapterOption {
	return func(a *Adapter) {
		a.digester = digester
	}
}

// WithArtifactVerifier sets a Verifier instead of using the default verification implementation.
// This is useful for testing purposes, e.g. to inject a mock Verifier that does not perform actual verification.
func WithArtifactVerifier(verifier Verifier) AdapterOption {
	return func(a *Adapter) {
		a.artifactVerifier = verifier
	}
}

// WithVectorVerifier sets a Verifier instead of using the default verification implementation.
// This is useful for testing purposes, e.g. to inject a mock Verifier that does not perform actual verification.
func WithVectorVerifier(verifier Verifier) AdapterOption {
	return func(a *Adapter) {
		a.vectorVerifier = verifier
	}
}

// WithDefaultArtifactVerificationAndTrustAnchor enables default verification of artifacts.
// It will use ArtifactSignatureName as the target signature name for verification.
// The required ctrl.Manager is used to set up an informer for watching a config map that contains a trust anchor for verification.
// Environment variables VerifierTrustAnchorConfigMapNameEnv and VerifierTrustAnchorConfigMapNamespaceEnv are required.
// If setup fails this option will panic.
func WithDefaultArtifactVerificationAndTrustAnchor(mgr ctrl.Manager) AdapterOption {
	return func(a *Adapter) {
		cfg, ns := os.Getenv(VerifierTrustAnchorConfigMapNameEnv), os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
		if cfg == "" || ns == "" {
			panic("ocm artifact verification: env vars OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME, OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE missing")
		}
		trustCfg := &RSAVerifierTrustAnchorConfig{configMapName: cfg, configMapNamespace: ns}
		v, err := NewRSAVerifier(mgr, trustCfg, ArtifactSignatureName)
		if err != nil {
			panic(fmt.Sprintf("unable to set up default ocm artifact verification: %v", err))
		}
		a.artifactVerifier = v
	}
}

// WithDefaultVectorVerificationAndTrustAnchor enables default signing and verification of vectors.
// It will use VectorAssemblySignatureName as the target signature name for signing and verification.
// The required ctrl.Manager is used to set up an informer for watching a config map that contains a trust anchor for verification.
// Environment variables VerifierTrustAnchorConfigMapNameEnv and VerifierTrustAnchorConfigMapNamespaceEnv are required.
// If setup fails this option will panic.
func WithDefaultVectorVerificationAndTrustAnchor(mgr ctrl.Manager) AdapterOption {
	return func(a *Adapter) {
		cfg, ns := os.Getenv(VerifierTrustAnchorConfigMapNameEnv), os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
		if cfg == "" || ns == "" {
			panic("ocm vector verification: env vars OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME, OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE missing")
		}
		trustCfg := &RSAVerifierTrustAnchorConfig{configMapName: cfg, configMapNamespace: ns}
		v, err := NewRSAVerifier(mgr, trustCfg, VectorAssemblySignatureName)
		if err != nil {
			panic(fmt.Sprintf("unable to set up default ocm vector verification: %v", err))
		}
		a.vectorVerifier = v
	}
}

// WithDefaultVectorSigning enables default signing of vectors.
// It will use VectorAssemblySignatureName as the target signature name for signing.
// The required ctrl.Manager is used to set up an informer for watching a secret that contains signing credentials.
// Environment variables CredentialSecretNameEnv and CredentialSecretNamespaceEnv are required.
// If setup fails this option will panic.
func WithDefaultVectorSigning(mgr ctrl.Manager) AdapterOption {
	return func(a *Adapter) {
		s, err := NewRSASigner(mgr, VectorAssemblySignatureName)
		if err != nil {
			panic(fmt.Sprintf("unable to set up default ocm vector signing: %v", err))
		}
		a.vectorSigner = s
	}
}
