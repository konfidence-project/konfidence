package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/ports"
	"github.com/konfidence-project/konfidence/pkg/lrucache"
)

const (
	defaultReconcileInterval         = 30 * time.Second
	deletionRequeueInterval          = 5 * time.Second
	stageConfigurationControllerName = "stage-configuration-controller"

	// A Stage lives in spec.targetNamespace, a different namespace than the
	// StageConfiguration, so ownerReference-based GC cannot delete it. The
	// finalizer lets the controller delete the Stage before releasing the object.
	stageConfigurationFinalizer = "konfidence.cloud/stage-configuration-finalizer"

	managedByLabelKey             = "app.kubernetes.io/managed-by"
	stageConfigurationRefLabelKey = "konfidence.cloud/stage-configuration"
)

// StageConfigurationReconciler reconciles a StageConfiguration object
type StageConfigurationReconciler struct {
	client.Client
	Recorder events.EventRecorder
	Cache    *lrucache.Cache[*konfidence.StageConfiguration, ports.VectorPort]
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stages,verbs=get;create;update;patch;delete
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

	if !stageConfiguration.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, stageConfiguration)
	}

	if controllerutil.AddFinalizer(stageConfiguration, stageConfigurationFinalizer) {
		if err := r.Update(ctx, stageConfiguration); err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to add finalizer to stageConfiguration: %w", err)
		}
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

	adapter, err := r.Cache.Lookup(ctx, r.Client, stageConfiguration)
	if err != nil {
		err = fmt.Errorf("building OCM clients: %w", err)
		r.updateStageConfigurationReadyStatus(stageConfiguration, false, err.Error())
		return err
	}

	vector, err := adapter.GetLatestVectorVersion(ctx, stageConfiguration.Spec.Vector)
	if err != nil {
		err = fmt.Errorf("unable to get vector component version %s: %w", stageConfiguration.Spec.Vector, err)
		r.updateStageConfigurationReadyStatus(stageConfiguration, false, err.Error())
		return err
	}

	stage, operationResult, err := r.createOrUpdateStage(ctx, stageConfiguration, vector)
	if err != nil {
		r.updateStageConfigurationReadyStatus(stageConfiguration, false, err.Error())
		return err
	}

	r.updateStageConfigurationReadyStatus(stageConfiguration, true, fmt.Sprintf("StageConfiguration %s reconciled", stageConfiguration.Name))

	// Only emit an event when the Stage actually changed
	if operationResult != controllerutil.OperationResultNone {
		msg := fmt.Sprintf("Stage %s/%s %s from StageConfiguration %s", stage.Namespace, stage.Name, operationResult, stageConfiguration.Name)
		r.Recorder.Eventf(stageConfiguration, nil, corev1.EventTypeNormal, "StageConfigurationReconciled", "StageConfigurationReconciled", msg)
		log.Info(msg)
	}
	return nil
}

// createOrUpdateStage writes the Stage into spec.targetNamespace with managed-by
// labels so the controller owns its lifecycle and won't touch a foreign Stage.
func (r *StageConfigurationReconciler) createOrUpdateStage(
	ctx context.Context,
	stageConfiguration *konfidence.StageConfiguration,
	vector string,
) (*konfidence.Stage, controllerutil.OperationResult, error) {
	stage := &konfidence.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageConfiguration.Spec.Name,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, stage, func() error {
		// CreationTimestamp is zero only on the create path; a set value means the
		// Stage pre-exists and must already be ours to be updated.
		if !stage.CreationTimestamp.IsZero() && !isManagedBy(stage, stageConfiguration) {
			return fmt.Errorf("stage %s/%s already exists and is not managed by StageConfiguration %s/%s",
				stage.Namespace, stage.Name, stageConfiguration.Namespace, stageConfiguration.Name)
		}
		if stage.Labels == nil {
			stage.Labels = map[string]string{}
		}
		stage.Labels[managedByLabelKey] = stageConfigurationControllerName
		stage.Labels[stageConfigurationRefLabelKey] = stageConfigurationRef(stageConfiguration)
		stage.Spec.Vector = vector
		return nil
	})
	if err != nil {
		return nil, operationResult, fmt.Errorf("failed to create or update stage: %w", err)
	}
	return stage, operationResult, nil
}

// reconcileDelete deletes the managed Stage (if any) and then releases the
// finalizer so the StageConfiguration can be garbage-collected.
func (r *StageConfigurationReconciler) reconcileDelete(
	ctx context.Context,
	stageConfiguration *konfidence.StageConfiguration,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(stageConfiguration, stageConfigurationFinalizer) {
		return ctrl.Result{}, nil
	}

	stage := &konfidence.Stage{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      stageConfiguration.Spec.Name,
		Namespace: stageConfiguration.Spec.TargetNamespace,
	}, stage)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("unable to fetch managed stage: %w", err)
	}

	// The Stage still exists and belongs to this StageConfiguration: issue the
	// delete and requeue until it is gone before releasing the finalizer.
	if err == nil && isManagedBy(stage, stageConfiguration) {
		if stage.DeletionTimestamp.IsZero() {
			if delErr := r.Delete(ctx, stage); delErr != nil && !apierrors.IsNotFound(delErr) {
				return ctrl.Result{}, fmt.Errorf("unable to delete managed stage: %w", delErr)
			}
		}
		log.Info("waiting for managed stage deletion", "stage", fmt.Sprintf("%s/%s", stage.Namespace, stage.Name))
		return ctrl.Result{RequeueAfter: deletionRequeueInterval}, nil
	}

	controllerutil.RemoveFinalizer(stageConfiguration, stageConfigurationFinalizer)
	if err := r.Update(ctx, stageConfiguration); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to remove finalizer from stageConfiguration: %w", err)
	}
	return ctrl.Result{}, nil
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

// stageConfigurationRef returns the managed-by label value for the owning
// StageConfiguration (namespace_name, sanitized to a valid label value).
func stageConfigurationRef(stageConfiguration *konfidence.StageConfiguration) string {
	return sanitizeLabelValue(stageConfiguration.Namespace + "_" + stageConfiguration.Name)
}

func isManagedBy(stage *konfidence.Stage, stageConfiguration *konfidence.StageConfiguration) bool {
	return stage.Labels[managedByLabelKey] == stageConfigurationControllerName &&
		stage.Labels[stageConfigurationRefLabelKey] == stageConfigurationRef(stageConfiguration)
}

// sanitizeLabelValue converts a string into a valid Kubernetes label value:
// '/' becomes '_', lowercased, and truncated to 63 characters.
func sanitizeLabelValue(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ToLower(s)
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.StageConfiguration{}, builder.WithPredicates(reconcilePredicate())).
		Named(stageConfigurationControllerName).
		Complete(r)
}

// reconcilePredicate triggers on spec (generation) changes and on deletion.
// GenerationChangedPredicate alone would drop the deletion event (setting a
// deletionTimestamp does not bump generation), leaving the finalizer stuck.
func reconcilePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() ||
				!e.ObjectNew.GetDeletionTimestamp().IsZero()
		},
	}
}

// NewStageConfigurationReconciler wires a StageConfigurationReconciler for the given manager.
func NewStageConfigurationReconciler(
	mgr ctrl.Manager,
	cache *lrucache.Cache[*konfidence.StageConfiguration, ports.VectorPort],
) *StageConfigurationReconciler {
	return &StageConfigurationReconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorder(stageConfigurationControllerName),
		Cache:    cache,
	}
}
