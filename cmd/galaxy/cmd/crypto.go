package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"k8s.io/apimachinery/pkg/types"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

// cryptoConfig holds all crypto dependencies resolved from env vars during startup.
type cryptoConfig struct {
	// VectorVerifier verifies vector assembly signatures. NoopVerifier if disabled.
	VectorVerifier crypto.Verifier
	// ArtifactVerifier verifies artifact signatures. NoopVerifier if disabled.
	ArtifactVerifier crypto.Verifier
	// VectorSigner signs assembled vectors. NoopSigner if disabled.
	VectorSigner crypto.Signer
}

// resolveCryptoConfig reads env vars and sets up all crypto providers and dependencies.
// It ensures the ConfigMapTrustAnchorProvider is only created once even if multiple
// concerns (artifact verification, vector verification) need it.
func resolveCryptoConfig(ctx context.Context, mgr mcmanager.Manager) (*cryptoConfig, error) {
	cfg := &cryptoConfig{
		VectorVerifier:   crypto.NoopVerifier{},
		ArtifactVerifier: crypto.NoopVerifier{},
		VectorSigner:     crypto.NoopSigner{},
	}

	verifyVector := parseBoolEnv(OcmVectorVerifyEnv)
	verifyArtifact := parseBoolEnv(OcmArtifactVerifyEnv)
	signVector := parseBoolEnv(OcmVectorSignEnv)

	needVectorVerification := verifyVector
	needArtifactVerification := verifyArtifact
	needTrustAnchor := needVectorVerification || needArtifactVerification
	needSigning := signVector

	// Set up shared trust anchor provider if any verification is needed
	var trustAnchorProvider *crypto.ConfigMapTrustAnchorProvider
	if needTrustAnchor {
		configMapName := os.Getenv(VerifierTrustAnchorConfigMapNameEnv)
		namespace := os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
		if configMapName == "" || namespace == "" {
			return nil, fmt.Errorf("env variables %s and/or %s not set",
				VerifierTrustAnchorConfigMapNameEnv, VerifierTrustAnchorConfigMapNamespaceEnv)
		}

		trustAnchorProvider = crypto.NewConfigMapTrustAnchorProvider(
			types.NamespacedName{Name: configMapName, Namespace: namespace})
		if err := trustAnchorProvider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
			return nil, fmt.Errorf("unable to set up config map trust anchor provider: %w", err)
		}
	}

	// Vector verification
	if needVectorVerification {
		setupLog.Info("OCM vector verification is enabled")
		rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.VectorAssemblySignature},
			crypto.WithCredentialProvider(trustAnchorProvider))
		if err != nil {
			return nil, fmt.Errorf("could not initialize RSA vector verifier: %w", err)
		}
		cfg.VectorVerifier = rsaVerifier
	} else {
		setupLog.Info("OCM vector verification is disabled")
	}

	// Artifact verification
	if needArtifactVerification {
		setupLog.Info("OCM artifact verification is enabled")
		rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.ArtifactSignature},
			crypto.WithCredentialProvider(trustAnchorProvider))
		if err != nil {
			return nil, fmt.Errorf("could not initialize RSA artifact verifier: %w", err)
		}
		cfg.ArtifactVerifier = rsaVerifier
	} else {
		setupLog.Info("OCM artifact verification is disabled")
	}

	// Vector signing
	if needSigning {
		setupLog.Info("OCM vector signing is enabled")
		secretName := os.Getenv(SigningCredentialSecretNameEnv)
		secretNamespace := os.Getenv(SigningCredentialSecretNamespaceEnv)
		if secretName == "" || secretNamespace == "" {
			return nil, fmt.Errorf("env variables %s and/or %s not set",
				SigningCredentialSecretNameEnv, SigningCredentialSecretNamespaceEnv)
		}

		secretProvider := crypto.NewSecretSigningCredentialsProvider(
			types.NamespacedName{Name: secretName, Namespace: secretNamespace})
		if err := secretProvider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
			return nil, fmt.Errorf("unable to set up secret signing credentials provider: %w", err)
		}

		rsaSigner, err := crypto.NewRSASigner(secretProvider, []string{crypto.VectorAssemblySignature})
		if err != nil {
			return nil, fmt.Errorf("could not initialize RSA vector signer: %w", err)
		}
		cfg.VectorSigner = rsaSigner
	} else {
		setupLog.Info("OCM vector signing is disabled")
	}

	return cfg, nil
}

// getVectorVerifier is a convenience helper for subcommands that only need vector verification.
func getVectorVerifier(ctx context.Context, mgr mcmanager.Manager) (crypto.Verifier, error) {
	if !parseBoolEnv(OcmVectorVerifyEnv) {
		setupLog.Info("OCM vector verification is disabled")
		return crypto.NoopVerifier{}, nil
	}

	setupLog.Info("OCM vector verification is enabled")
	configMapName := os.Getenv(VerifierTrustAnchorConfigMapNameEnv)
	namespace := os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
	if configMapName == "" || namespace == "" {
		return nil, fmt.Errorf("env variables %s and/or %s not set",
			VerifierTrustAnchorConfigMapNameEnv, VerifierTrustAnchorConfigMapNamespaceEnv)
	}

	provider := crypto.NewConfigMapTrustAnchorProvider(types.NamespacedName{Name: configMapName, Namespace: namespace})
	if err := provider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
		return nil, fmt.Errorf("unable to set up config map trust anchor provider: %w", err)
	}

	rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.VectorAssemblySignature},
		crypto.WithCredentialProvider(provider))
	if err != nil {
		return nil, fmt.Errorf("could not initialize RSA vector verifier: %w", err)
	}

	return rsaVerifier, nil
}

// parseBoolEnv reads an env var and returns true only if it parses to true.
func parseBoolEnv(envVar string) bool {
	val := strings.ToLower(os.Getenv(envVar))
	if val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false
	}
	return b
}
