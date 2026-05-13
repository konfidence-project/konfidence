/*
Copyright 2025.

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

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"github.com/konfidence-project/landscape-vector-deployment-controller/internal/controller"
	"github.com/konfidence-project/landscape-vector-deployment-controller/pkg/ocm"
	"github.com/konfidence-project/pkg/ocm/crypto"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	// +kubebuilder:scaffold:imports
)

const (
	OcmArtifactVerifyEnv                     = "OCM_ARTIFACT_VERIFY"
	OcmVectorVerifyEnv                       = "OCM_VECTOR_VERIFY"
	VerifierTrustAnchorConfigMapNameEnv      = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME"
	VerifierTrustAnchorConfigMapNamespaceEnv = "OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(landscape.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

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
		LeaderElectionID:       "a67b73e3.konfidence.cloud",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	var vectorVerifier, artifactVerifier crypto.Verifier = crypto.NoopVerifier{}, crypto.NoopVerifier{}
	verifyArtifactEnv := strings.ToLower(os.Getenv(OcmArtifactVerifyEnv))
	verifyVectorEnv := strings.ToLower(os.Getenv(OcmVectorVerifyEnv))

	if verifyArtifactEnv != "" || verifyVectorEnv != "" {
		verifyArtifact, err := strconv.ParseBool(verifyArtifactEnv)
		if err != nil {
			verifyArtifact = false
		}
		verifyVector, err := strconv.ParseBool(verifyVectorEnv)
		if err != nil {
			verifyVector = false
		}

		if verifyArtifact || verifyVector {
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

			if verifyArtifact {
				setupLog.Info("OCM artifact verification is enabled")
				rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.ArtifactSignature},
					crypto.WithCredentialProvider(configMapProvider))
				if err != nil {
					setupLog.Error(err, "Could not initialize artifact RSA signature verifier")
					os.Exit(1)
				}

				artifactVerifier = rsaVerifier
			} else {
				setupLog.Info("OCM artifact verification is disabled")
			}

			if verifyVector {
				setupLog.Info("OCM vector verification is enabled")
				rsaVerifier, err := crypto.NewRSAVerifier([]string{crypto.VectorAssemblySignature},
					crypto.WithCredentialProvider(configMapProvider))
				if err != nil {
					setupLog.Error(err, "Could not initialize vector RSA signature verifier")
					os.Exit(1)
				}

				vectorVerifier = rsaVerifier
			} else {
				setupLog.Info("OCM vector verification is disabled")
			}
		}
	} else {
		setupLog.Info("OCM vector and artifact verification is disabled")
	}

	secret, err := resolveRegistryCredentials(ctx, mgr)
	if err != nil {
		setupLog.Error(err, "unable to load registry credentials secret")
		os.Exit(1)
	}

	ocmAdapter, err := ocm.NewOcmAdapter(ctx, secret, vectorVerifier, artifactVerifier)

	if err != nil {
		setupLog.Error(err, "unable to create OCM adapter")
		os.Exit(1)
	}
	if err := (&controller.VectorDeploymentReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorder(controller.VectorDeploymentControllerName),
		OcmAdapter: ocmAdapter,
	}).SetupWithManager(mgr, "vectordeployment"); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VectorDeployment")
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
