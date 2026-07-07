package vectordeployment

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ocmcredentials "ocm.software/open-component-model/bindings/go/credentials"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/konfidence-project/konfidence/internal/star/vectordeployment/internal/controller"
	"github.com/konfidence-project/konfidence/internal/star/vectordeployment/internal/ocm"
	pkgcredentials "github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
)

const OperatorFlagName = "VectorDeployment"

const (
	VectorSignaturesEnv      = "KONFIDENCE_DEPLOYMENT_VECTOR_SIGNATURES"
	ArtifactSignaturesEnv    = "KONFIDENCE_DEPLOYMENT_ARTIFACT_SIGNATURES"
	CredentialsSecretNameEnv = "KONFIDENCE_DEPLOYMENT_CREDENTIALS_SECRET_NAME"
	CredentialsSecretNsEnv   = "KONFIDENCE_DEPLOYMENT_CREDENTIALS_SECRET_NAMESPACE"
)

// Options configures the vector deployment controllers.
type Options struct {
	// OCISecret is an optional Secret holding OCI credentials (.ocmconfig or .dockerconfigjson).
	// Flat-merged with the credentials Secret from env vars into one resolver.
	OCISecret *corev1.Secret

	// Limiter bounds process-wide CPU-bound crypto work across both verifiers.
	// Required; use crypto.NewLimiter(0) for GOMAXPROCS.
	Limiter crypto.Limiter
}

// SetupControllers registers all vector deployment controllers with the given manager.
func SetupControllers(ctx context.Context, mgr manager.Manager, logger logr.Logger, opts Options) error {
	if opts.Limiter == nil {
		return fmt.Errorf("setup: Limiter is required; use crypto.NewLimiter(0) for GOMAXPROCS")
	}

	log := logf.FromContext(ctx)

	resolver, err := buildResolver(ctx, mgr, opts.OCISecret)
	if err != nil {
		return fmt.Errorf("building credential resolver: %w", err)
	}

	vectorVerifier, artifactVerifier, err := buildVerifiers(log, resolver, opts.Limiter)
	if err != nil {
		return fmt.Errorf("building crypto verifiers: %w", err)
	}

	ocmAdapter, err := ocm.NewAdapter(ctx, resolver, vectorVerifier, artifactVerifier)
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

// buildVerifiers constructs process-wide verifiers from environment variables.
// Absent signature env vars produce a NoopVerifier for that slot.
// The provided resolver is shared by both verifiers; nil means system trust roots.
func buildVerifiers(log logr.Logger, resolver ocmcredentials.Resolver, limiter crypto.Limiter) (vectorVerifier, artifactVerifier crypto.Verifier, err error) {
	vectorNames := parseSignatureNames(VectorSignaturesEnv)
	artifactNames := parseSignatureNames(ArtifactSignaturesEnv)

	vectorVerifier, err = buildVerifier(log, resolver, limiter, vectorNames, "vector")
	if err != nil {
		return nil, nil, err
	}

	artifactVerifier, err = buildVerifier(log, resolver, limiter, artifactNames, "artifact")
	if err != nil {
		return nil, nil, err
	}

	return vectorVerifier, artifactVerifier, nil
}

func buildVerifier(log logr.Logger, resolver ocmcredentials.Resolver, limiter crypto.Limiter, names []string, kind string) (crypto.Verifier, error) {
	if len(names) == 0 {
		log.Info(kind + " verification disabled")
		return crypto.NoopVerifier{}, nil
	}

	v, err := crypto.NewVerifierBuilder().
		WithSpecs(makeSpecs(names)).
		WithResolver(resolver).
		WithLimiter(limiter).
		WithLogger(log).
		Build()
	if err != nil {
		return nil, fmt.Errorf("build %s verifier: %w", kind, err)
	}

	log.Info(kind+" verification enabled", "signatures", names)
	return v, nil
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
