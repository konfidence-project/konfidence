/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kcp-dev/multicluster-provider/apiexport"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	vectorpromotion "github.com/konfidence-project/konfidence/internal/galaxy/vector-promotion"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
)

var vectorpromotionCmd = &cobra.Command{
	Use:   "vectorpromotion",
	Short: "Start the vector promotion controllers",
	Long:  `Starts the VectorPromotion, VectorPromotionTTL, and VectorPromotionStatusPropagation controllers.`,
	RunE:  startVectorPromotion,
}

func startVectorPromotion(cmd *cobra.Command, args []string) error {
	cfg := ctrl.GetConfigOrDie()
	leaderElectionCfg := cfg
	if kubernetesServiceHost != "" && kubernetesServicePort != 0 {
		inClusterCfg, err := rest.InClusterConfig()
		if err != nil {
			setupLog.Error(err, "unable to get in-cluster config for leader election")
			return err
		}

		leaderElectionCfg = inClusterCfg
	}

	var err error
	var provider multicluster.Provider
	if kcpEndpointSlice != "" {
		provider, err = apiexport.New(cfg, kcpEndpointSlice, apiexport.Options{Scheme: scheme, Log: &setupLog})
		if err != nil {
			setupLog.Error(err, "unable to construct cluster provider")
			return err
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
		return err
	}

	ctx := ctrl.SetupSignalHandler()

	opts := vectorpromotion.Options{}
	verifyVectorEnv := strings.ToLower(os.Getenv(OcmVectorVerifyEnv))
	if verifyVectorEnv != "" {
		verifyVector, err := strconv.ParseBool(verifyVectorEnv)
		if err != nil {
			return fmt.Errorf("unable to parse env variable %q into bool: %w", OcmVectorVerifyEnv, err)
		}

		if verifyVector {
			configMapName, namespace := os.Getenv(VerifierTrustAnchorConfigMapNameEnv),
				os.Getenv(VerifierTrustAnchorConfigMapNamespaceEnv)
			if configMapName == "" || namespace == "" {
				return fmt.Errorf("env variables %s and/or %s not set", VerifierTrustAnchorConfigMapNameEnv,
					VerifierTrustAnchorConfigMapNamespaceEnv)
			}

			configMapProvider := crypto.NewConfigMapTrustAnchorProvider(
				types.NamespacedName{Name: configMapName, Namespace: namespace})
			if err = configMapProvider.SetupWithManager(ctx, mgr.GetLocalManager()); err != nil {
				setupLog.Error(err, "unable to set up config map trust anchor provider")
				return err
			}

			opts.VectorVerificationProvider = configMapProvider
		} else {
			setupLog.Info("OCM vector verification is disabled")
		}
	}

	if err := vectorpromotion.SetupControllers(ctx, mgr, scheme, opts); err != nil {
		setupLog.Error(err, "unable to set up vector promotion controllers")
		return err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(vectorpromotionCmd)
}
