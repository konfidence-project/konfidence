package cmd

import (
	"os"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
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
	metricsAddr          string
	controllersSpec      string
	leaseID              string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "konfidence",
	Short: "Run the Konfidence operator",
	Long: `konfidence is the Konfidence operator.

It runs a controller manager that reconciles all Konfidence API resources
on a target Kubernetes cluster.`,
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
	utilruntime.Must(konfidence.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	loggerOpts := zap.Options{
		Development: true,
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&loggerOpts)))

	rootCmd.PersistentFlags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to. Use \"0\" to disable.")
	rootCmd.PersistentFlags().BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	rootCmd.Flags().StringVar(&controllersSpec, "controllers", "*",
		"Comma-separated glob expression selecting which controllers to enable. "+
			"Examples: '*' (all), 'Stage', '!VectorAssembly,*' (all except), 'Vector*'. "+
			"Tokens are set-based and order-independent; '!' negates.")
	rootCmd.Flags().StringVar(&leaseID, "lease-id", "konfidence-operator.konfidence.cloud", "The ID used for leader election.")
}
