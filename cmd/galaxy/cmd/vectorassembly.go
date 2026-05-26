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
package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/multicluster-runtime/pkg/multicluster"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/kcp-dev/multicluster-provider/apiexport"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	vectorassembly "github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly"
	pkgcmd "github.com/konfidence-project/konfidence/pkg/cmd"
)

var vectorassemblyCmd = &cobra.Command{
	Use:   "vectorassembly",
	Short: "Start the vector assembly controllers",
	Long:  `Starts the VectorTemplate controller for assembling vectors from templates.`,
	RunE:  startVectorAssemblyController,
}

func startVectorAssemblyController(cmd *cobra.Command, args []string) error {
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
		LeaderElectionID:       leaseID,
		LeaderElectionConfig:   leaderElectionCfg,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	ctx := ctrl.SetupSignalHandler()

	cryptoCfg, err := pkgcmd.ResolveCryptoConfig(ctx, mgr.GetLocalManager(), setupLog)
	if err != nil {
		return err
	}

	if err := vectorassembly.SetupControllers(mgr, scheme, vectorassembly.Options{
		ArtifactVerifier: cryptoCfg.ArtifactVerifier,
		VectorVerifier:   cryptoCfg.VectorVerifier,
		VectorSigner:     cryptoCfg.VectorSigner,
	}); err != nil {
		setupLog.Error(err, "unable to set up vector assembly controllers")
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
	vectorassemblyCmd.Flags().StringVar(&leaseID, "lease-id", "galaxy-vectorassembly.konfidence.cloud",
		"The ID used for leader election.")
	rootCmd.AddCommand(vectorassemblyCmd)
}
