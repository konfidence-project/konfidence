package vectordeployment

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ocmcredentials "ocm.software/open-component-model/bindings/go/credentials"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/controller"
	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/ocm"
	pkgcredentials "github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "VectorDeployment"

const (
	VectorSignaturesEnv      = "KONFIDENCE_DEPLOYMENT_VECTOR_SIGNATURES"
	ArtifactSignaturesEnv    = "KONFIDENCE_DEPLOYMENT_ARTIFACT_SIGNATURES"
	CredentialsSecretNameEnv = "KONFIDENCE_DEPLOYMENT_CREDENTIALS_SECRET_NAME"
	CredentialsSecretNsEnv   = "KONFIDENCE_DEPLOYMENT_CREDENTIALS_SECRET_NAMESPACE"
)

// Domain wires the vector deployment controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "VectorDeployment",
		Setup: func(ctx context.Context, deps operator.Deps) error {
			registrySecret, err := resolveRegistryCredentials(ctx, deps.Mgr)
			if err != nil {
				return fmt.Errorf("load registry credentials secret: %w", err)
			}
			return SetupControllers(ctx, deps.Mgr, deps.Logger, Options{
				OCISecret: registrySecret,
				Verifier:  deps.Verifier,
			})
		},
	}
}

// resolveRegistryCredentials loads the optional registry-credentials secret;
// nil means no secret is configured.
func resolveRegistryCredentials(ctx context.Context, mgr manager.Manager) (*corev1.Secret, error) {
	const secretName = "registry-credentials"
	const secretNamespace = "konfidence-system"

	secret := &corev1.Secret{}
	err := mgr.GetAPIReader().Get(ctx, types.NamespacedName{Namespace: secretNamespace, Name: secretName}, secret)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", secretNamespace, secretName, err)
	}
	return secret, nil
}

// Options configures the vector deployment controllers.
type Options struct {
	// OCISecret is an optional Secret holding OCI credentials (.ocmconfig or .dockerconfigjson).
	// Flat-merged with the credentials Secret from env vars into one resolver.
	OCISecret *corev1.Secret

	// Verifier is the shared process-wide OCM verifier. Required; usually from operator.Deps.Verifier.
	Verifier crypto.Verifier
}

// SetupControllers registers all vector deployment controllers with the given manager.
func SetupControllers(ctx context.Context, mgr manager.Manager, logger logr.Logger, opts Options) error {
	if opts.Verifier == nil {
		return fmt.Errorf("setup: Verifier is required; usually from operator.Deps.Verifier")
	}

	log := logf.FromContext(ctx)

	resolver, err := buildResolver(ctx, mgr, opts.OCISecret)
	if err != nil {
		return fmt.Errorf("building credential resolver: %w", err)
	}

	vectorSpecs := loadSpecs(log, VectorSignaturesEnv, "vector")
	artifactSpecs := loadSpecs(log, ArtifactSignaturesEnv, "artifact")

	ocmAdapter, err := ocm.NewAdapter(ctx, resolver, opts.Verifier, vectorSpecs, artifactSpecs)
	if err != nil {
		return fmt.Errorf("unable to create OCM adapter: %w", err)
	}

	if err := (&controller.VectorDeploymentReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorder(controller.VectorDeploymentControllerName),
		OcmAdapter: ocmAdapter,
	}).SetupWithManager(mgr, "vectordeployment"); err != nil {
		logger.Error(err, "unable to create controller", "controller", "VectorDeployment")
		return err
	}
	return nil
}

// buildResolver constructs a single flat-merged credentials.Resolver from the credentials
// Secret named by env vars and the optional OCISecret. Returns nil if neither is configured.
func buildResolver(ctx context.Context, mgr manager.Manager, ociSecret *corev1.Secret) (ocmcredentials.Resolver, error) {
	secretName := os.Getenv(CredentialsSecretNameEnv)
	secretNs := os.Getenv(CredentialsSecretNsEnv)

	switch {
	case secretName != "" && secretNs == "":
		return nil, fmt.Errorf("%s is set but %s is missing", CredentialsSecretNameEnv, CredentialsSecretNsEnv)
	case secretName == "" && secretNs != "":
		return nil, fmt.Errorf("%s is set but %s is missing", CredentialsSecretNsEnv, CredentialsSecretNameEnv)
	}

	var refs []pkgcredentials.Ref
	if secretName != "" {
		refs = append(refs, pkgcredentials.Ref{Name: secretName})
	}
	if ociSecret != nil {
		refs = append(refs, pkgcredentials.Ref{Name: ociSecret.Name})
	}

	namespace := secretNs
	if namespace == "" && ociSecret != nil {
		namespace = ociSecret.Namespace
	}

	return pkgcredentials.ResolverFromRefs(ctx, mgr.GetAPIReader(), namespace, refs)
}

// loadSpecs reads a comma-separated list of signature names from envVar and
// converts them to SignatureSpecs. An empty env var produces an empty slice,
// which the shared Verifier no-ops on — the "verification disabled" state.
func loadSpecs(log logr.Logger, envVar, kind string) []crypto.SignatureSpec {
	names := parseSignatureNames(envVar)
	if len(names) == 0 {
		log.Info(kind + " verification disabled")
		return nil
	}
	log.Info(kind+" verification enabled", "signatures", names)
	return makeSpecs(names)
}

func makeSpecs(names []string) []crypto.SignatureSpec {
	specs := make([]crypto.SignatureSpec, len(names))
	for i, name := range names {
		specs[i] = crypto.DefaultSignatureSpec(name, nil)
	}
	return specs
}

func parseSignatureNames(envVar string) []string {
	val := strings.TrimSpace(os.Getenv(envVar))
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		if n := strings.TrimSpace(p); n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}
