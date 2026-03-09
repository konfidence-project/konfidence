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

	"github.com/konfidence-project/crds/api/global/v1alpha1"
	"github.com/konfidence-project/pkg/ocm/crypto"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/konfidence-project/gcp-vector-assembly-controller/pkg/ocm"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller"
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
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "48fb26ce.konfidence.cloud",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	var adapterConfig []ocm.AdapterOption
	verifyArtifact := strings.ToLower(os.Getenv(OcmArtifactVerifyEnv))
	verifyAndSignVector := strings.ToLower(os.Getenv(OcmVectorSignAndVerifyEnv))

	if verifyArtifact != "" || verifyAndSignVector != "" {
		configMapName, namespace := os.Getenv(VerifierTrustAnchorConfigMapNameEnv),
			os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
		if configMapName == "" || namespace == "" {
			setupLog.Error(fmt.Errorf("env variables %s and/or %s not set", VerifierTrustAnchorConfigMapNameEnv,
				VerifierTrustAnchorConfigMapNamespaceEnv), "")
			os.Exit(1)
		}

		configMapProvider := crypto.NewConfigMapTrustAnchorProvider(types.NamespacedName{Name: configMapName, Namespace: namespace})
		if err = configMapProvider.SetupWithManager(ctx, mgr); err != nil {
			setupLog.Error(err, "unable to set up config map trust anchor provider")
			os.Exit(1)
		}

		chk, err := strconv.ParseBool(verifyArtifact)
		if err == nil && chk {
			adapterConfig = append(adapterConfig, ocm.WithDefaultArtifactVerification(configMapProvider))
		} else {
			setupLog.Info("OCM artifact verification is disabled")
		}

		chk, err = strconv.ParseBool(verifyAndSignVector)
		if err == nil && chk {
			secretName, secretNamespace := os.Getenv(SigningCredentialSecretNameEnv), os.Getenv(SigningCredentialSecretNamespaceEnv)
			if secretName == "" || secretNamespace == "" {
				setupLog.Error(fmt.Errorf("env variables %s and/or %s not set", SigningCredentialSecretNameEnv,
					SigningCredentialSecretNamespaceEnv), "")
				os.Exit(1)
			}

			secretProvider := crypto.NewSecretSigningCredentialsProvider(types.NamespacedName{Name: secretName, Namespace: secretNamespace})
			if err = secretProvider.SetupWithManager(ctx, mgr); err != nil {
				setupLog.Error(err, "unable to set up secret signing credentials provider")
				os.Exit(1)
			}

			adapterConfig = append(adapterConfig,
				ocm.WithDefaultVectorSigning(secretProvider),
				ocm.WithDefaultVectorVerification(configMapProvider))
		} else {
			setupLog.Info("OCM vector signing and verification is disabled")
		}
	}

	// read secret from a k8s secret.
	// todo: in future, we want configure the credentials for accessing OCI registries in a controller-specific configuration.
	secret, err := loadSecret(ctx, mgr)
	if err != nil {
		setupLog.Error(err, "unable to load registry credentials secret")
		os.Exit(1)
	}
	adapterConfig = append(adapterConfig, ocm.WithOcmClient(ctx, secret))

	if err := (&controller.VectorTemplateReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		OcmAdapter: ocm.NewAdapter(adapterConfig...),
		Recorder:   mgr.GetEventRecorder(controller.VectorAssemblyControllerName),
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

// loadSecret loads the registry credentials secret from the k8s cluster.
// returns nil if the secret is not found.
func loadSecret(ctx context.Context, mgr manager.Manager) (*v1.Secret, error) {
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
