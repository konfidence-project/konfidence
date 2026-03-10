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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/konfidence-project/gcp-stage-configuration-controller/pkg/ocm"
	"github.com/konfidence-project/pkg/ocm/crypto"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
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
	KubernetesServiceHost                    = "KUBERNETES_SERVICE_HOST"
	KubernetesServicePort                    = "KUBERNETES_SERVICE_PORT"
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
	var endpointSlice string
	var tempSkipOciRegistry bool

	flag.StringVar(&endpointSlice, "endpointslice", "",
		"Set the APIExportEndpointSlice name to watch")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&tempSkipOciRegistry, "skip-oci", false,
		"Temporarily skip resolving of OCI vector and use default version instead")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	cfg := ctrl.GetConfigOrDie()
	leaderElectionCfg := cfg
	serviceHost, servicePort := os.Getenv(KubernetesServiceHost), os.Getenv(KubernetesServicePort)
	if serviceHost != "" && servicePort != "" {
		inClusterCfg, err := rest.InClusterConfig()
		if err != nil {
			setupLog.Error(err, "unable to get in-cluster config for leader election")
			os.Exit(1)
		}

		leaderElectionCfg = inClusterCfg
	}

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

	context := ctrl.SetupSignalHandler()
	var vectorVerifier crypto.Verifier
	if strings.ToLower(os.Getenv(OcmVectorVerifyEnv)) != "true" {
		setupLog.Info("ocm vector verification is disabled")
		vectorVerifier = crypto.NoopVerifier{}
	} else {
		configMapName, namespace := os.Getenv(VerifierTrustAnchorConfigMapNameEnv),
			os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
		if configMapName == "" || namespace == "" {
			setupLog.Error(fmt.Errorf("env variables %s and/or %s not set", VerifierTrustAnchorConfigMapNameEnv,
				VerifierTrustAnchorConfigMapNamespaceEnv), "")
			os.Exit(1)
		}

		provider := crypto.NewConfigMapTrustAnchorProvider(types.NamespacedName{Name: configMapName, Namespace: namespace})
		if err = provider.SetupWithManager(context, mgr.GetLocalManager()); err != nil {
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
	}

	if err := (&controller.StageConfigurationReconciler{
		Mgr:        mgr,
		OCMClient:  ocm.OCIClient{VectorVerifier: vectorVerifier},
		Scheme:     scheme,
		RestConfig: cfg,
		SkipOci:    tempSkipOciRegistry,
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
	if err := mgr.Start(context); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
