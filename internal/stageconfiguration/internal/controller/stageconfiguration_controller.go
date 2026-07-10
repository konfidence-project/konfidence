package controller

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/ports"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/template"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
)

const (
	defaultReconcileInterval         = 30 * time.Second
	stageConfigurationControllerName = "stage-configuration-controller"
	clusterMarker                    = "/clusters/"
	clusterPattern                   = "\\/clusters\\/[^/]+$"
)

var clusterRegex = regexp.MustCompile(clusterPattern)

// StageConfigurationReconciler reconciles a StageConfiguration object
type StageConfigurationReconciler struct {
	Mgr        mcmanager.Manager
	Cache      *clientcache.Cache[*galaxy.StageConfiguration, ports.VectorPort]
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
}

func NewStageConfigurationReconciler(
	mgr mcmanager.Manager,
	scheme *runtime.Scheme,
	restConfig *rest.Config,
	cache *clientcache.Cache[*galaxy.StageConfiguration, ports.VectorPort],
) *StageConfigurationReconciler {
	return &StageConfigurationReconciler{
		Mgr:        mgr,
		Cache:      cache,
		Scheme:     scheme,
		RestConfig: restConfig,
	}
}

// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=stageconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=stageconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=stagesyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=stagesyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *StageConfigurationReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stageConfiguration started...")
	log.Info(fmt.Sprintf("Cluster: %s", req.ClusterName))

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	clusterClient := cluster.GetClient()
	recorder := cluster.GetEventRecorder(stageConfigurationControllerName)

	// get stageConfiguration
	stageConfiguration := &galaxy.StageConfiguration{}
	if err := clusterClient.Get(ctx, req.NamespacedName, stageConfiguration); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStageConfiguration := stageConfiguration.DeepCopy()
	err = r.reconcileStageConfiguration(ctx, clusterClient, req.ClusterName, stageConfiguration, recorder)

	err = pkgctrl.PatchStatusIfChanged(
		ctx,
		clusterClient,
		stageConfiguration,
		originalStageConfiguration,
		stageConfiguration.Status,
		originalStageConfiguration.Status,
		"unable to update stageConfiguration status",
		err,
		"an error occurred while reconciling stageConfiguration",
	)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: defaultReconcileInterval}, nil
}

func (r *StageConfigurationReconciler) reconcileStageConfiguration(
	ctx context.Context,
	clusterClient client.Client,
	clusterName string,
	stageConfiguration *galaxy.StageConfiguration,
	recorder events.EventRecorder,
) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stageConfiguration")
	r.updateStageConfigurationReadyStatus(stageConfiguration, false, "")

	adapter, err := r.Cache.Lookup(ctx, clusterClient, clusterName, stageConfiguration)
	if err != nil {
		return fmt.Errorf("building OCM clients: %w", err)
	}

	// get vector with specific version or alias
	vector, err := adapter.GetLatestVectorVersion(ctx, stageConfiguration.Spec.Vector)
	if err != nil {
		return fmt.Errorf("unable to get vector component version %s: %w", stageConfiguration.Spec.Vector, err)
	}

	// write to cluster or workspace
	var targetClient client.Client
	if r.Mgr.GetProvider() == nil {
		targetClient = clusterClient
	} else {
		// copy the default kubeconfig and change the server url
		targetCfg := rest.CopyConfig(r.RestConfig)

		targetWorkspaceHost, err := r.getTargetWorkspaceHost(targetCfg.Host, stageConfiguration)
		if err != nil {
			return fmt.Errorf("unable to get target workspace host: %w", err)
		}

		targetCfg.Host = targetWorkspaceHost

		// create a new client for the target cluster
		cl, err := client.New(targetCfg, client.Options{
			Scheme: r.Scheme,
		})
		if err != nil {
			return fmt.Errorf("could not create target client from rest config %w", err)
		}

		targetClient = cl
	}

	// create or update stageSync
	stageSync, operationResult, err := r.createOrUpdateStageSync(ctx, targetClient, stageConfiguration, vector)
	if err != nil {
		return err
	}

	// mark stageConfiguration as ready
	r.updateStageConfigurationReadyStatus(stageConfiguration, true, fmt.Sprintf("StageConfiguration %s reconciled", stageConfiguration.Name))

	msg := fmt.Sprintf("StageSync %s %s with StageConfiguration %s", stageSync.Name, operationResult, stageConfiguration.Name)
	recorder.Eventf(stageConfiguration, nil, corev1.EventTypeNormal, "StageConfigurationReconciled", "StageConfigurationReconciled", msg)
	log.Info(msg)
	return nil
}

func (r *StageConfigurationReconciler) createOrUpdateStageSync(
	ctx context.Context, targetClient client.Client,
	stageConfiguration *galaxy.StageConfiguration, vector string,
) (*galaxy.StageSync, controllerutil.OperationResult, error) {
	stageSync, stageTemplateBytes, err := r.constructStageSync(stageConfiguration, vector)
	if err != nil {
		return nil, controllerutil.OperationResultNone, err
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, targetClient, stageSync, func() error {
		var originalTemplate, stageTemplate template.StageTemplate
		if err := json.Unmarshal(stageSync.Spec.StageTemplate.Raw, &originalTemplate); err != nil {
			return err
		}
		if err := json.Unmarshal(stageTemplateBytes, &stageTemplate); err != nil {
			return err
		}

		if !reflect.DeepEqual(originalTemplate, stageTemplate) {
			stageSync.Spec.StageTemplate.Raw = stageTemplateBytes
		}

		return nil
	})
	if err != nil {
		return nil, operationResult, fmt.Errorf("failed to create or update stageSync: %w", err)
	}
	return stageSync, operationResult, nil
}

func (r *StageConfigurationReconciler) constructStageSync(stageConfiguration *galaxy.StageConfiguration, vector string) (*galaxy.StageSync, []byte, error) {
	stageTemplate := r.constructStageTemplate(stageConfiguration, vector)
	stageTemplateJSON, err := json.Marshal(stageTemplate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal stage template: %w", err)
	}

	// TODO this might lead to naming collisions. to resolve this issue one
	// TODO could e.g. use a digest that is computed using stage name and tenant
	stageSyncName := fmt.Sprintf("sync-%s", stageConfiguration.Spec.Name)

	return &galaxy.StageSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageSyncName,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
		Spec: galaxy.StageSyncSpec{
			StageTemplate: runtime.RawExtension{Raw: stageTemplateJSON},
		},
	}, stageTemplateJSON, nil
}

func (r *StageConfigurationReconciler) constructStageTemplate(stageConfiguration *galaxy.StageConfiguration, vector string) *template.StageTemplate {
	// TODO replace APIVersion with a configured or determined value
	return &template.StageTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       star.StageKind,
			APIVersion: "star.konfidence.cloud/v1alpha1",
		},
		Metadata: template.NamespacedName{
			Name:      stageConfiguration.Spec.Name,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
		Spec: star.StageSpec{
			Vector: vector,
		},
	}
}

func (r *StageConfigurationReconciler) getTargetWorkspaceHost(host string, stageConfiguration *galaxy.StageConfiguration) (string, error) {
	// if kcp is used the target workspace is mandatory
	if stageConfiguration.Spec.TargetWorkspace == nil || len(*stageConfiguration.Spec.TargetWorkspace) == 0 {
		return "", fmt.Errorf("stage configuration does not contain a target workspace")
	}

	if !clusterRegex.MatchString(host) {
		return "", fmt.Errorf("could not match clusters entry at end of config host %s", host)
	}

	separatorIdx := strings.LastIndex(host, clusterMarker)
	if separatorIdx == -1 {
		return "", fmt.Errorf("missing clusters entry in config host %s", host)
	}

	return host[:separatorIdx] + clusterMarker + *stageConfiguration.Spec.TargetWorkspace, nil
}

func (r *StageConfigurationReconciler) updateStageConfigurationReadyStatus(stageConfiguration *galaxy.StageConfiguration, ready bool, message string) {
	status := metav1.ConditionFalse
	if ready {
		status = metav1.ConditionTrue
	}

	meta.SetStatusCondition(&stageConfiguration.Status.Conditions, metav1.Condition{
		Type:               galaxy.StageConfigurationReadyCondition,
		Status:             status,
		Reason:             galaxy.StageConfigurationReadyCondition,
		Message:            message,
		ObservedGeneration: stageConfiguration.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageConfigurationReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		For(&galaxy.StageConfiguration{}, mcbuilder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named(stageConfigurationControllerName).
		Complete(r)
}
