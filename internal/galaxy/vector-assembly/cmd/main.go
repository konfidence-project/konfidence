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

	"github.com/kcp-dev/multicluster-provider/apiexport"
	apisv1alpha1 "github.com/kcp-dev/sdk/apis/apis/v1alpha1"
	apisv1alpha2 "github.com/kcp-dev/sdk/apis/apis/v1alpha2"
	corev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	tenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/internal/controller/domain"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/pkg/ocm"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/internal/controller"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	OcmArtifactVerifyEnv                     = "OCM_ARTIFACT_VERIFY"
	OcmVectorSignAndVerifyEnv                = "OCM_VECTOR_SIGN_AND_VERIFY"
	VerifierTrustAnchorConfigMapNameEnv      = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME"
	VerifierTrustAnchorConfigMapNamespaceEnv = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE"
	SigningCredentialSecretNameEnv           = "OCM_RSA_SIGNING_KEY_SECRET_NAME"
	SigningCredentialSecretNamespaceEnv      = "OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE"
	KubernetesServiceHost                    = "KUBERNETES_SERVICE_HOST"
	KubernetesServicePort                    = "KUBERNETES_SERVICE_PORT"
	KcpEndpointSlice                         = "KCP_ENDPOINT_SLICE"
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
	serviceHost, servicePort := os.Getenv(KubernetesServiceHost), os.Getenv(KubernetesServicePort)
	if serviceHost != "" && servicePort != "" {
		inClusterCfg, err := rest.InClusterConfig()
		if err != nil {
			setupLog.Error(err, "unable to get in-cluster config for leader election")
			os.Exit(1)
		}

		leaderElectionCfg = inClusterCfg
	}

	endpointSlice := os.Getenv(KcpEndpointSlice)

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
		LeaderElectionID:       "48fb26ce.konfidence.cloud",
		LeaderElectionConfig:   leaderElectionCfg,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	var adapterConfig []ocm.AdapterOption
	verifyArtifactEnv := strings.ToLower(os.Getenv(OcmArtifactVerifyEnv))
	verifyAndSignVectorEnv := strings.ToLower(os.Getenv(OcmVectorSignAndVerifyEnv))

	if verifyArtifactEnv != "" || verifyAndSignVectorEnv != "" {
		verifyArtifact, err := strconv.ParseBool(verifyArtifactEnv)
		if err != nil {
			verifyArtifact = false
		}
		verifyAndSignVector, err := strconv.ParseBool(verifyAndSignVectorEnv)
		if err != nil {
			verifyAndSignVector = false
		}

		if verifyArtifact || verifyAndSignVector {
			configMapName, namespace := os.Getenv(VerifierTrustAnchorConfigMapNameEnv),
				os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
			if configMapName == "" || namespace == "" {
				setupLog.Error(fmt.Errorf("env variables %s and/or %s not set", VerifierTrustAnchorConfigMapNameEnv,
					VerifierTrustAnchorConfigMapNamespaceEnv), "")
				os.Exit(1)
			}

			configMapProvider := crypto.NewConfigMapTrustAnchorProvider(types.NamespacedName{Name: configMapName, Namespace: namespace})
			if err = configMapProvider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
				setupLog.Error(err, "unable to set up config map trust anchor provider")
				os.Exit(1)
			}

			if verifyArtifact {
				adapterConfig = append(adapterConfig, ocm.WithDefaultArtifactVerification(configMapProvider))
			} else {
				setupLog.Info("OCM artifact verification is disabled")
			}

			if verifyAndSignVector {
				secretName, secretNamespace := os.Getenv(SigningCredentialSecretNameEnv), os.Getenv(SigningCredentialSecretNamespaceEnv)
				if secretName == "" || secretNamespace == "" {
					setupLog.Error(fmt.Errorf("env variables %s and/or %s not set", SigningCredentialSecretNameEnv,
						SigningCredentialSecretNamespaceEnv), "")
					os.Exit(1)
				}

				secretProvider := crypto.NewSecretSigningCredentialsProvider(types.NamespacedName{Name: secretName, Namespace: secretNamespace})
				if err = secretProvider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
					setupLog.Error(err, "unable to set up secret signing credentials provider")
					os.Exit(1)
				}

				adapterConfig = append(adapterConfig,
					ocm.WithDefaultVectorSigning(secretProvider),
					ocm.WithDefaultVectorVerification(configMapProvider))
			}
		} else {
			setupLog.Info("OCM vector signing and verification is disabled")
		}
	}

	if err := (&controller.VectorTemplateReconciler{
		Mgr:                   mgr,
		Scheme:                scheme,
		OcmClientProvider:     repository.DefaultOciClientProvider,
		VectorOcmPortProvider: ocm.NewPortProvider(adapterConfig...),
		VersionGenerator:      domain.TimestampVectorVersionGenerator,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VectorTemplate")
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
