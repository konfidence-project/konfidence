package controller

import (
	"context"
	"fmt"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/ports"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
)

const (
	defaultReconcileInterval         = 30 * time.Second
	stageConfigurationControllerName = "stage-configuration-controller"
)

// StageConfigurationReconciler reconciles a StageConfiguration object
type StageConfigurationReconciler struct {
	client.Client
	Recorder events.EventRecorder
	Cache    *clientcache.Cache[*konfidence.StageConfiguration, ports.VectorPort]
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *StageConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stageConfiguration started...")

	stageConfiguration := &konfidence.StageConfiguration{}
	if err := r.Get(ctx, req.NamespacedName, stageConfiguration); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStageConfiguration := stageConfiguration.DeepCopy()
	err := r.reconcileStageConfiguration(ctx, stageConfiguration)

	err = pkgctrl.PatchStatusIfChanged(
		ctx,
		r.Client,
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
	stageConfiguration *konfidence.StageConfiguration,
) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stageConfiguration")
	r.updateStageConfigurationReadyStatus(stageConfiguration, false, "")

	adapter, err := r.Cache.Lookup(ctx, r.Client, stageConfiguration)
	if err != nil {
		return fmt.Errorf("building OCM clients: %w", err)
	}

	vector, err := adapter.GetLatestVectorVersion(ctx, stageConfiguration.Spec.Vector)
	if err != nil {
		return fmt.Errorf("unable to get vector component version %s: %w", stageConfiguration.Spec.Vector, err)
	}

	stage, operationResult, err := r.createOrUpdateStage(ctx, stageConfiguration, vector)
	if err != nil {
		return err
	}

	r.updateStageConfigurationReadyStatus(stageConfiguration, true, fmt.Sprintf("StageConfiguration %s reconciled", stageConfiguration.Name))

	msg := fmt.Sprintf("Stage %s %s with StageConfiguration %s", stage.Name, operationResult, stageConfiguration.Name)
	r.Recorder.Eventf(stageConfiguration, nil, corev1.EventTypeNormal, "StageConfigurationReconciled", "StageConfigurationReconciled", msg)
	log.Info(msg)
	return nil
}

func (r *StageConfigurationReconciler) createOrUpdateStage(
	ctx context.Context,
	stageConfiguration *konfidence.StageConfiguration,
	vector string,
) (*konfidence.Stage, controllerutil.OperationResult, error) {
	// TODO this might lead to naming collisions. to resolve this issue one
	// TODO could e.g. use a digest that is computed using stage name and tenant
	stage := &konfidence.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageConfiguration.Spec.Name,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, stage, func() error {
		stage.Spec.Vector = vector
		return nil
	})
	if err != nil {
		return nil, operationResult, fmt.Errorf("failed to create or update stage: %w", err)
	}
	return stage, operationResult, nil
}

func (r *StageConfigurationReconciler) updateStageConfigurationReadyStatus(stageConfiguration *konfidence.StageConfiguration, ready bool, message string) {
	status := metav1.ConditionFalse
	if ready {
		status = metav1.ConditionTrue
	}

	meta.SetStatusCondition(&stageConfiguration.Status.Conditions, metav1.Condition{
		Type:               konfidence.StageConfigurationReadyCondition,
		Status:             status,
		Reason:             konfidence.StageConfigurationReadyCondition,
		Message:            message,
		ObservedGeneration: stageConfiguration.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.StageConfiguration{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&konfidence.Stage{},
			handler.EnqueueRequestsFromMapFunc(r.findStageConfigurationsForStage),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Named(stageConfigurationControllerName).
		Complete(r)
}

// findStageConfigurationsForStage maps a Stage back to the StageConfiguration(s)
// that own it so drifted or deleted Stages get reconciled. The Stage lives in
// Spec.TargetNamespace, which may differ from the StageConfiguration namespace,
// so cross-namespace owner references aren't usable and we match on name +
// target namespace instead.
func (r *StageConfigurationReconciler) findStageConfigurationsForStage(ctx context.Context, obj client.Object) []reconcile.Request {
	stageConfigurations := &konfidence.StageConfigurationList{}
	if err := r.List(ctx, stageConfigurations); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for i := range stageConfigurations.Items {
		sc := &stageConfigurations.Items[i]
		if sc.Spec.Name == obj.GetName() && sc.Spec.TargetNamespace == obj.GetNamespace() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: sc.Name, Namespace: sc.Namespace},
			})
		}
	}
	return requests
}

// NewStageConfigurationReconciler wires a StageConfigurationReconciler for the given manager.
func NewStageConfigurationReconciler(
	mgr ctrl.Manager,
	cache *clientcache.Cache[*konfidence.StageConfiguration, ports.VectorPort],
) *StageConfigurationReconciler {
	return &StageConfigurationReconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorder(stageConfigurationControllerName),
		Cache:    cache,
	}
}
