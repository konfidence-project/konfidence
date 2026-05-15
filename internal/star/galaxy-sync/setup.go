package galaxysync

import (
	"fmt"

	"github.com/go-logr/logr"
	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/star/galaxy-sync/internal/config"
	"github.com/konfidence-project/konfidence/internal/star/galaxy-sync/internal/controller"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

// Options holds the dependencies for the galaxy-sync controller domain.
type Options struct {
	// ControllerNamespace is the namespace the controller is running in.
	ControllerNamespace string
}

// SetupControllers wires up the galaxy-sync controllers with the given manager.
func SetupControllers(mgr ctrl.Manager, logger logr.Logger, scheme *runtime.Scheme, opts Options) error {
	// Use a direct (non-cached) client so the Secret can be fetched before
	// the manager's cache is started.
	directClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("unable to create direct client for kubeconfig Secret lookup: %w", err)
	}

	namespace := opts.ControllerNamespace
	if namespace == "" {
		namespace = "default"
	}

	remoteConfig, err := config.FromSecret(directClient, namespace)
	if err != nil {
		return fmt.Errorf("unable to resolve remote kubeconfig: %w", err)
	}

	// remoteCluster is the cluster the controller watches StageSync objects on.
	// Default to the local cluster (single-cluster mode).
	var remoteCluster cluster.Cluster = mgr

	if remoteConfig != nil {
		// Multi-cluster: build a dedicated cluster for the remote (Galaxy) side.
		logger.Info("Remote kubeconfig found; running in multi-cluster mode",
			"kubeconfig-secret", fmt.Sprintf("%s/%s", namespace, config.SecretName),
			"secret-key", config.SecretKey,
			"remote-cluster-server", remoteConfig.Host)

		remoteCluster, err = cluster.New(remoteConfig, func(options *cluster.Options) {
			options.Scheme = scheme
			// Restrict the remote cache to only StageSync objects.
			// The remote cluster is a kcp workspace where StageSync is served
			// via an APIBinding — restricting the cache prevents discovery
			// errors caused by trying to watch resource types that are not
			// available on that workspace.
			options.Cache.ByObject = map[client.Object]cache.ByObject{
				&global.StageSync{}: {},
			}
		})
		if err != nil {
			return fmt.Errorf("unable to create remote cluster: %w", err)
		}
		// Adding the cluster to the manager ensures its cache is started and
		// fully synced before the controller's informers are set up.
		if err := mgr.Add(remoteCluster); err != nil {
			return fmt.Errorf("unable to add remote cluster to manager: %w", err)
		}
	} else {
		logger.Info("No remote kubeconfig Secret found; running in single-cluster mode")
	}

	if err := (&controller.StageSyncReconciler{
		LocalClient:   mgr.GetClient(),
		RemoteCluster: remoteCluster,
		Scheme:        scheme,
		Recorder:      remoteCluster.GetEventRecorder(controller.StageSyncControllerName),
		LandscapeName: config.LandscapeName(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create StageSyncReconciler: %w", err)
	}

	return nil
}
