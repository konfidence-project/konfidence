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

func getVectorVerifier(
	ctx context.Context,
	mgr mcmanager.Manager,
) (crypto.Verifier, error) {

	var vectorVerifier crypto.Verifier = crypto.NoopVerifier{}
	verifyVectorEnv := strings.ToLower(os.Getenv(OcmVectorVerifyEnv))
	if verifyVectorEnv == "" {
		setupLog.Info("OCM vector verification is disabled")
		return vectorVerifier, nil
	}
	verifyVector, err := strconv.ParseBool(verifyVectorEnv)
	if err != nil {
		verifyVector = false
	}

	if !verifyVector {
		setupLog.Info("OCM vector verification is disabled")
		return vectorVerifier, nil
	}
	setupLog.Info("OCM vector verification is enabled")
	configMapName, namespace := os.Getenv(VerifierTrustAnchorConfigMapNameEnv),
		os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
	if configMapName == "" || namespace == "" {
		setupLog.Error(fmt.Errorf("env variables %s and/or %s not set", VerifierTrustAnchorConfigMapNameEnv,
			VerifierTrustAnchorConfigMapNamespaceEnv), "")
		return nil, err
	}

	provider := crypto.NewConfigMapTrustAnchorProvider(types.NamespacedName{Name: configMapName, Namespace: namespace})

	if err = provider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
		setupLog.Error(err, "unable to set up config map trust anchor provider")
		return nil, err
	}

	rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.VectorAssemblySignature},
		crypto.WithCredentialProvider(provider))
	if err != nil {
		setupLog.Error(err, "Could not initialize RSA signature verifier")
		return nil, err
	}

	return rsaVerifier, nil
}
