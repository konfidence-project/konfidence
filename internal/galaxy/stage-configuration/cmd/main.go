/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/konfidence-project/gcp-stage-configuration-controller/pkg/ocm"
	"github.com/konfidence-project/pkg/ocm/crypto"
	pkgOcm "github.com/konfidence-project/pkg/ocm/repository"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kcp-dev/multicluster-provider/apiexport"
	apisv1alpha1 "github.com/kcp-dev/sdk/apis/apis/v1alpha1"
	apisv1alpha2 "github.com/kcp-dev/sdk/apis/apis/v1alpha2"
	corev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	tenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	"github.com/konfidence-project/gcp-stage-configuration-controller/internal/controller"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	// +kubebuilder:scaffold:imports
)

const (
	OcmVectorVerifyEnv                       = "OCM_VECTOR_VERIFY"
	VerifierTrustAnchorConfigMapNameEnv      = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME"
	VerifierTrustAnchorConfigMapNamespaceEnv = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE"
	KubernetesServiceHostEnv                 = "KUBERNETES_SERVICE_HOST"
	KubernetesServicePortEnv                 = "KUBERNETES_SERVICE_PORT"
	KcpEndpointSliceEnv                      = "KCP_ENDPOINT_SLICE"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(apisv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apisv1alpha2.AddToScheme(scheme))
	utilruntime.Must(corev1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenancyv1alpha1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(common.AddToScheme(scheme))
	utilruntime.Must(global.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	cfg := ctrl.GetConfigOrDie()
	leaderElectionCfg := cfg
	serviceHost, servicePort := os.Getenv(KubernetesServiceHostEnv), os.Getenv(KubernetesServicePortEnv)
	if serviceHost != "" && servicePort != "" {
		inClusterCfg, err := rest.InClusterConfig()
		if err != nil {
			setupLog.Error(err, "unable to get in-cluster config for leader election")
			os.Exit(1)
		}

		leaderElectionCfg = inClusterCfg
	}

	endpointSlice := os.Getenv(KcpEndpointSliceEnv)

	var err error
	var provider multicluster.Provider = nil
	if endpointSlice != "" {
		provider, err = apiexport.New(cfg, endpointSlice, apiexport.Options{Scheme: scheme, Log: &setupLog})
		if err != nil {
			setupLog.Error(err, "unable to construct cluster provider")
			os.Exit(1)
		}
	}

	mgr, err := mcmanager.New(cfg, provider, ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "969492ce.konfidence.cloud",
		LeaderElectionConfig:   leaderElectionCfg,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	var vectorVerifier crypto.Verifier = crypto.NoopVerifier{}
	verifyVectorEnv := strings.ToLower(os.Getenv(OcmVectorVerifyEnv))
	if verifyVectorEnv != "" {
		verifyVector, err := strconv.ParseBool(verifyVectorEnv)
		if err != nil {
			verifyVector = false
		}

		if verifyVector {
			setupLog.Info("OCM vector verification is enabled")
			configMapName, namespace := os.Getenv(VerifierTrustAnchorConfigMapNameEnv),
				os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
			if configMapName == "" || namespace == "" {
				setupLog.Error(fmt.Errorf("env variables %s and/or %s not set", VerifierTrustAnchorConfigMapNameEnv,
					VerifierTrustAnchorConfigMapNamespaceEnv), "")
				os.Exit(1)
			}

			provider := crypto.NewConfigMapTrustAnchorProvider(types.NamespacedName{Name: configMapName, Namespace: namespace})
			if err = provider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
				setupLog.Error(err, "unable to set up config map trust anchor provider")
				os.Exit(1)
			}

			rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.VectorAssemblySignature},
				crypto.WithCredentialProvider(provider))
			if err != nil {
				setupLog.Error(err, "Could not initialize RSA signature verifier")
				os.Exit(1)
			}

			vectorVerifier = rsaVerifier
		} else {
			setupLog.Info("OCM vector verification is disabled")
		}
	} else {
		setupLog.Info("OCM vector verification is disabled")
	}

	registryCredentials, err := resolveRegistryCredentials(ctx, mgr.GetLocalManager())
	if err != nil {
		setupLog.Error(err, "unable to resolve registry credentials", "controller", "StageConfiguration")
		os.Exit(1)
	}

	ocmClient, err := pkgOcm.NewOciClientBuilder().WithLogger(ctrl.Log).
		WithDockerConfigJsonSecret(registryCredentials).Build(ctx)
	if err != nil {
		setupLog.Error(err, "unable to create pkg ocm client", "controller", "StageConfiguration")
		os.Exit(1)
	}

	if err := (&controller.StageConfigurationReconciler{
		Mgr:        mgr,
		VectorPort: ocm.VectorOCMAdapter{VectorVerifier: vectorVerifier, OcmClient: ocmClient},
		Scheme:     scheme,
		RestConfig: cfg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StageConfiguration")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// TODO: the credentials for accessing OCI registries should be configured in a controller-specific configuration.
// resolveRegistryCredentials loads the registry credentials secret from the k8s cluster.
// returns nil if the secret is not found.
func resolveRegistryCredentials(ctx context.Context, mgr manager.Manager) (*v1.Secret, error) {
	const secretName = "registry-credentials"
	const secretNamespace = "konfidence-system"

	secret := &v1.Secret{}
	err := mgr.GetAPIReader().Get(ctx, types.NamespacedName{Namespace: secretNamespace, Name: secretName}, secret)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", secretNamespace, secretName, err)
	}

	return secret, nil
}
