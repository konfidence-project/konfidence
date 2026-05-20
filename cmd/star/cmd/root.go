/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/spf13/cobra"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var (
	enableLeaderElection bool
	probeAddr            string
	controllersSpec      string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "star",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: startOperator,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	utilruntime.Must(landscape.AddToScheme(scheme))
	utilruntime.Must(global.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	loggerOpts := zap.Options{
		Development: true,
	}

	// fs := pflag.NewFlagSet("example", pflag.ExitOnError)
	// fs.AddGoFlagSet()

	// loggerOpts.BindFlags(fs)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&loggerOpts)))

	rootCmd.PersistentFlags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	rootCmd.PersistentFlags().BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	rootCmd.Flags().StringVar(&controllersSpec, "controllers", "*",
		"Comma-separated glob expression selecting which controllers to enable. "+
			"Examples: '*' (all), 'Stage', '!Stage,*' (all except), 'Vector*'. "+
			"Tokens are set-based and order-independent; '!' negates.")
}
