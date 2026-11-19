package vectordeployment

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/controller"
	vdocm "github.com/konfidence-project/konfidence/internal/vectordeployment/internal/ocm"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	pkgcredentials "github.com/konfidence-project/konfidence/pkg/ocm/credentials"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "VectorDeployment"

const (
	MaxArtifactManifestDownloadsPerVectorDeploymentEnv = "KONFIDENCE_MAX_ARTIFACT_MANIFEST_DOWNLOADS_PER_VECTOR_DEPLOYMENT"
	MaxGlobalArtifactManifestDownloadsEnv              = "KONFIDENCE_MAX_GLOBAL_ARTIFACT_MANIFEST_DOWNLOADS"
)

// Domain wires the vector deployment controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "VectorDeployment",
		Setup: func(ctx context.Context, deps operator.Deps) error {
			return SetupControllers(ctx, deps.Mgr, deps.Logger, Options{
				Verifier: deps.Verifier,
			})
		},
	}
}

// Options configures the vector deployment controllers.
type Options struct {
	// Verifier is the shared process-wide OCM verifier. Required; usually from operator.Deps.Verifier.
	Verifier crypto.Verifier

	// MaxArtifactManifestDownloadsPerVectorDeployment is the maximum number of parallel artifact manifest downloads per vector deployment.
	MaxArtifactManifestDownloadsPerVectorDeployment int

	// MaxGlobalArtifactManifestDownloads is the maximum number of parallel artifact manifest downloads across all vector deployments per controller.
	MaxGlobalArtifactManifestDownloads int
}

// SetupControllers registers all vector deployment controllers with the given manager.
func SetupControllers(ctx context.Context, mgr manager.Manager, logger logr.Logger, opts Options) error {
	if opts.Verifier == nil {
		return fmt.Errorf("setup: Verifier is required; usually from operator.Deps.Verifier")
	}

	manifestDownloadsPerDeployment, err := parseManifestDownloadLimit(
		opts.MaxArtifactManifestDownloadsPerVectorDeployment,
		MaxArtifactManifestDownloadsPerVectorDeploymentEnv,
		controller.DefaultArtifactManifestDownloadsPerVectorDeployment,
	)
	if err != nil {
		return err
	}

	globalManifestDownloads, err := parseManifestDownloadLimit(
		opts.MaxGlobalArtifactManifestDownloads,
		MaxGlobalArtifactManifestDownloadsEnv,
		controller.DefaultGlobalArtifactManifestDownloads,
	)
	if err != nil {
		return err
	}

	verifier := opts.Verifier
	k8sReader := mgr.GetClient()

	if err := (&controller.VectorDeploymentReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(controller.VectorDeploymentControllerName),
		AdapterFactory: func(ctx context.Context, vd *konfidence.VectorDeployment) (controller.VectorOcmPort, error) {
			ns := &corev1.Namespace{}
			if err := k8sReader.Get(ctx, types.NamespacedName{Name: vd.Namespace}, ns); err != nil {
				return nil, fmt.Errorf("get namespace %q: %w", vd.Namespace, err)
			}
			projectName := ns.Labels[pkgctrl.ProjectNameLabel]
			if projectName == "" {
				return nil, fmt.Errorf("namespace %q is missing label %s", vd.Namespace, pkgctrl.ProjectNameLabel)
			}
			project := &konfidence.Project{}
			if err := k8sReader.Get(ctx, types.NamespacedName{Name: projectName}, project); err != nil {
				return nil, fmt.Errorf("get project %q: %w", projectName, err)
			}
			projectNamespace := project.Status.Namespace
			if projectNamespace == "" {
				return nil, fmt.Errorf("project %q has no namespace configured", projectName)
			}
			resolver, err := pkgcredentials.ResolverFromCredentials(ctx, k8sReader, projectNamespace, project.Spec.Credentials)
			if err != nil {
				return nil, fmt.Errorf("resolve OCM credentials for project %q: %w", projectName, err)
			}
			ocmClient, err := pkgocm.NewOciClientBuilder().
				WithLogger(logger).
				WithResolver(resolver).
				Build(ctx)
			if err != nil {
				return nil, fmt.Errorf("build OCI client for project %q: %w", projectName, err)
			}
			return vdocm.NewOCMAdapter(ocmClient, resolver, verifier,
				crypto.SpecsFromVerify(project.Spec.VerifyVector),
				crypto.SpecsFromVerify(project.Spec.VerifyArtifacts),
			), nil
		},
		MaxArtifactManifestDownloadsPerDeployment: manifestDownloadsPerDeployment,
		MaxGlobalArtifactManifestDownloads:        globalManifestDownloads,
	}).SetupWithManager(mgr, "vectordeployment"); err != nil {
		logger.Error(err, "unable to create controller", "controller", "VectorDeployment")
		return err
	}

	logger.Info("configured artifact manifest downloads",
		"manifestDownloadsPerDeployment", manifestDownloadsPerDeployment,
		"globalManifestDownloads", globalManifestDownloads,
	)

	return nil
}

func parseManifestDownloadLimit(option int, envVar string, fallback int) (int, error) {
	value := option
	if value == 0 {
		raw := strings.TrimSpace(os.Getenv(envVar))
		if raw == "" {
			value = fallback
		} else {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				return 0, fmt.Errorf("%s must be an integer: %w", envVar, err)
			}
			value = parsed
		}
	}

	if value < 1 || value > controller.MaxParallelArtifactManifestDownloads {
		return 0, fmt.Errorf("%s must be between 1 and %d, got %d", envVar, controller.MaxParallelArtifactManifestDownloads, value)
	}

	return value, nil
}
