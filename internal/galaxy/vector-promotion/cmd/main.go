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
	"strconv"
	"strings"

	apisv1alpha1 "github.com/kcp-dev/sdk/apis/apis/v1alpha1"
	apisv1alpha2 "github.com/kcp-dev/sdk/apis/apis/v1alpha2"
	corev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	tenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	"github.com/konfidence-project/pkg/ocm/crypto"
	"github.com/konfidence-project/pkg/ocm/repository"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	"github.com/kcp-dev/multicluster-provider/apiexport"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"github.com/konfidence-project/gcp-vector-promotion-controller/internal/controller"
	"github.com/konfidence-project/gcp-vector-promotion-controller/internal/controller/ocm"
	// +kubebuilder:scaffold:imports
)

const (
	KubernetesServiceHostEnv                 = "KUBERNETES_SERVICE_HOST"
	KubernetesServicePortEnv                 = "KUBERNETES_SERVICE_PORT"
	KcpEndpointSliceEnv                      = "KCP_ENDPOINT_SLICE"
	OcmVectorVerifyEnv                       = "OCM_VECTOR_VERIFY"
	VerifierTrustAnchorConfigMapNameEnv      = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME"
	VerifierTrustAnchorConfigMapNamespaceEnv = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE"
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
	var provider multicluster.Provider
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
		LeaderElectionID:       "e83e292c.konfidence.cloud",
		LeaderElectionConfig:   leaderElectionCfg,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()

	var promotionAdapterConfig []ocm.PromotionAdapterOption
	verifyVectorEnv := strings.ToLower(os.Getenv(OcmVectorVerifyEnv))
	if verifyVectorEnv != "" {
		verifyVector, err := strconv.ParseBool(verifyVectorEnv)
		if err != nil {
			setupLog.Error(err, fmt.Sprintf("unable to parse env variable %q into bool", OcmVectorVerifyEnv))
			os.Exit(1)
		}

		if verifyVector {
			configMapName, namespace := os.Getenv(VerifierTrustAnchorConfigMapNameEnv),
				os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
			if configMapName == "" || namespace == "" {
				setupLog.Error(fmt.Errorf("env variables %s and/or %s not set", VerifierTrustAnchorConfigMapNameEnv,
					VerifierTrustAnchorConfigMapNamespaceEnv), "")
				os.Exit(1)
			}

			configMapProvider := crypto.NewConfigMapTrustAnchorProvider(
				types.NamespacedName{Name: configMapName, Namespace: namespace})
			if err = configMapProvider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
				setupLog.Error(err, "unable to set up config map trust anchor provider")
				os.Exit(1)
			}

			promotionAdapterConfig = append(promotionAdapterConfig, ocm.WithDefaultVectorVerification(configMapProvider))
		} else {
			setupLog.Info("OCM vector verification is disabled")
		}
	}

	if err := (&controller.VectorPromotionReconciler{
		Mgr:               mgr,
		Scheme:            scheme,
		PortProvider:      ocm.NewPromotionPortProvider(promotionAdapterConfig...),
		OcmClientProvider: repository.DefaultOciClientProvider,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VectorPromotion")
		os.Exit(1)
	}

	if err := (&controller.VectorPromotionTTLReconciler{
		Mgr:    mgr,
		Scheme: scheme,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VectorPromotionTTL")
		os.Exit(1)
	}

	if err := (&controller.VectorPromotionStatusPropagationReconciler{
		Mgr:    mgr,
		Scheme: scheme,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VectorPromotionStatusPropagation")
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
