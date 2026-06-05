package cmd

import (
	"os"
	"strconv"

	apisv1alpha1 "github.com/kcp-dev/sdk/apis/apis/v1alpha1"
	apisv1alpha2 "github.com/kcp-dev/sdk/apis/apis/v1alpha2"
	corev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	tenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	KubernetesServiceHostEnv = "KUBERNETES_SERVICE_HOST"
	KubernetesServicePortEnv = "KUBERNETES_SERVICE_PORT"
	KcpEndpointSliceEnv      = "KCP_ENDPOINT_SLICE"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var (
	enableLeaderElection  bool
	probeAddr             string
	metricsAddr           string
	kubernetesServiceHost string
	kubernetesServicePort int
	kcpEndpointSlice      string
	controllersSpec       string
	leaseID               string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "galaxy",
	Short: "Run the Konfidence galaxy control-plane operator",
	Long: `galaxy is the Konfidence control-plane operator.

It runs a multicluster controller manager with several controllers that reconcile galaxy API resources.
When a KCP APIExportEndpointSlice is provided it serves workspaces via KCP;
otherwise it operates against a single Kubernetes cluster.`,
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
	utilruntime.Must(apisv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apisv1alpha2.AddToScheme(scheme))
	utilruntime.Must(corev1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenancyv1alpha1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(star.AddToScheme(scheme))
	utilruntime.Must(galaxy.AddToScheme(scheme))

	loggerOpts := zap.Options{
		Development: true,
	}

	// fs := pflag.NewFlagSet("example", pflag.ExitOnError)
	// fs.AddGoFlagSet()

	// loggerOpts.BindFlags(fs)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&loggerOpts)))

	rootCmd.PersistentFlags().StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	rootCmd.PersistentFlags().StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to. Use \"0\" to disable.")
	rootCmd.PersistentFlags().BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	rootCmd.PersistentFlags().StringVar(&kcpEndpointSlice, "kcp-endpoint-slice-name", os.Getenv(KcpEndpointSliceEnv),
		"Provides a reference to the APIExport in KCP. If using single cluster mode this shall be empty.")
	rootCmd.PersistentFlags().StringVar(&kubernetesServiceHost, "kubernetes-host", os.Getenv(KubernetesServiceHostEnv),
		"connection host towards an out of band Kubernetes cluster")
	servicePort := os.Getenv(KubernetesServicePortEnv)
	var defaultPort int
	if servicePort != "" {
		defaultPort, _ = strconv.Atoi(servicePort)
	}
	rootCmd.PersistentFlags().IntVar(&kubernetesServicePort, "kubernetes-port", defaultPort, "connection port towards an out of band Kubernetes cluster")
	rootCmd.Flags().StringVar(&controllersSpec, "controllers", "*",
		"Comma-separated glob expression selecting which controllers to enable. "+
			"Examples: '*' (all), 'VectorAssembly', '!VectorAssembly,*' (all except), 'Vector*'. "+
			"Tokens are set-based and order-independent; '!' negates.")
	rootCmd.Flags().StringVar(&leaseID, "lease-id", "galaxy-operator.konfidence.cloud", "The ID used for leader election.")
}
