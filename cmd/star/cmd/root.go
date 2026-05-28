package cmd

import (
	"os"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
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
	leaseID              string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "star",
	Short: "Run the Konfidence star workload-cluster operator",
	Long: `star is the Konfidence workload-cluster operator.

It runs a controller manager with several controllers that reconcile star API resources on a target Kubernetes cluster.`,
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
	utilruntime.Must(star.AddToScheme(scheme))
	utilruntime.Must(galaxy.AddToScheme(scheme))
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
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	rootCmd.Flags().StringVar(&controllersSpec, "controllers", "*",
		"Comma-separated glob expression selecting which controllers to enable. "+
			"Examples: '*' (all), 'Stage', '!Stage,*' (all except), 'Vector*'. "+
			"Tokens are set-based and order-independent; '!' negates.")
	rootCmd.Flags().StringVar(&leaseID, "lease-id", "star-operator.konfidence.cloud", "The ID used for leader election.")
}
