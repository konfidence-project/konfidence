package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	VectorVerifyEnv                  = "OCM_VECTOR_VERIFY"
	ArtifactVerifyEnv                = "OCM_ARTIFACT_VERIFY"
	VectorSignEnv                    = "OCM_VECTOR_SIGN"
	VerifierTrustAnchorConfigMapName = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME"
	VerifierTrustAnchorConfigMapNs   = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE"
	SigningCredentialSecretName      = "OCM_RSA_SIGNING_KEY_SECRET_NAME"
	SigningCredentialSecretNs        = "OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE"
)

// CryptoConfig holds all crypto dependencies resolved from env vars during startup.
type CryptoConfig struct {
	// VectorVerifier verifies vector assembly signatures. NoopVerifier if disabled.
	VectorVerifier crypto.Verifier
	// ArtifactVerifier verifies artifact signatures. NoopVerifier if disabled.
	ArtifactVerifier crypto.Verifier
	// VectorSigner signs assembled vectors. NoopSigner if disabled.
	VectorSigner crypto.Signer
}

// ResolveCryptoConfig reads env vars and sets up all crypto providers, registering watches with the given manager.
func ResolveCryptoConfig(ctx context.Context, mgr ctrl.Manager, logger logr.Logger) (*CryptoConfig, error) {
	cfg := &CryptoConfig{
		VectorVerifier:   crypto.NoopVerifier{},
		ArtifactVerifier: crypto.NoopVerifier{},
		VectorSigner:     crypto.NoopSigner{},
	}

	verifyVector := ParseBoolEnv(VectorVerifyEnv)
	verifyArtifact := ParseBoolEnv(ArtifactVerifyEnv)
	signVector := ParseBoolEnv(VectorSignEnv)

	needTrustAnchor := verifyVector || verifyArtifact

	// Set up shared trust anchor provider if any verification is needed
	var trustAnchorProvider *crypto.ConfigMapTrustAnchorProvider
	if needTrustAnchor {
		configMapName := os.Getenv(VerifierTrustAnchorConfigMapName)
		namespace := os.Getenv(VerifierTrustAnchorConfigMapNs)
		if configMapName == "" || namespace == "" {
			return nil, fmt.Errorf("env variables %s and/or %s not set",
				VerifierTrustAnchorConfigMapName, VerifierTrustAnchorConfigMapNs)
		}

		trustAnchorProvider = crypto.NewConfigMapTrustAnchorProvider(
			types.NamespacedName{Name: configMapName, Namespace: namespace})
		if err := trustAnchorProvider.SetupWithManager(ctx, mgr); err != nil {
			return nil, fmt.Errorf("unable to set up config map trust anchor provider: %w", err)
		}
	}

	// Vector verification
	if verifyVector {
		logger.Info("OCM vector verification is enabled")
		rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.VectorAssemblySignature},
			crypto.WithCredentialProvider(trustAnchorProvider))
		if err != nil {
			return nil, fmt.Errorf("could not initialize RSA vector verifier: %w", err)
		}
		cfg.VectorVerifier = rsaVerifier
	} else {
		logger.Info("OCM vector verification is disabled")
	}

	// Artifact verification
	if verifyArtifact {
		logger.Info("OCM artifact verification is enabled")
		rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.ArtifactSignature},
			crypto.WithCredentialProvider(trustAnchorProvider))
		if err != nil {
			return nil, fmt.Errorf("could not initialize RSA artifact verifier: %w", err)
		}
		cfg.ArtifactVerifier = rsaVerifier
	} else {
		logger.Info("OCM artifact verification is disabled")
	}

	// Vector signing
	if signVector {
		logger.Info("OCM vector signing is enabled")
		secretName := os.Getenv(SigningCredentialSecretName)
		secretNamespace := os.Getenv(SigningCredentialSecretNs)
		if secretName == "" || secretNamespace == "" {
			return nil, fmt.Errorf("env variables %s and/or %s not set",
				SigningCredentialSecretName, SigningCredentialSecretNs)
		}

		secretProvider := crypto.NewSecretSigningCredentialsProvider(
			types.NamespacedName{Name: secretName, Namespace: secretNamespace})
		if err := secretProvider.SetupWithManager(ctx, mgr); err != nil {
			return nil, fmt.Errorf("unable to set up secret signing credentials provider: %w", err)
		}

		rsaSigner, err := crypto.NewRSASigner(secretProvider, []string{crypto.VectorAssemblySignature})
		if err != nil {
			return nil, fmt.Errorf("could not initialize RSA vector signer: %w", err)
		}
		cfg.VectorSigner = rsaSigner
	} else {
		logger.Info("OCM vector signing is disabled")
	}

	return cfg, nil
}

// ParseBoolEnv reads an env var and returns true only if it parses to true.
func ParseBoolEnv(envVar string) bool {
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
