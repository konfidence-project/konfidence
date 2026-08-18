package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
)

const StageVersionControllerName = "stage-version-controller"

// StageVersionReconciler reconciles a StageVersion object
type StageVersionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageversions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectormigrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectormigrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectoractivations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectoractivations/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *StageVersionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stageVersion started...")

	// get stageVersion
	stageVersion := &konfidence.StageVersion{}
	if err := r.Get(ctx, req.NamespacedName, stageVersion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStageVersion := stageVersion.DeepCopy()
	err := r.reconcileStageVersion(ctx, stageVersion)

	return ctrl.Result{}, pkgctrl.PatchStatusIfChanged(
		ctx,
		r.Client,
		stageVersion,
		originalStageVersion,
		stageVersion.Status,
		originalStageVersion.Status,
		"unable to update stageVersion status",
		err,
		"an error occurred while reconciling stageVersion",
	)
}

func (r *StageVersionReconciler) reconcileStageVersion(ctx context.Context, stageVersion *konfidence.StageVersion) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stageVersion")

	stageName, ok := stageVersion.Labels[pkgctrl.StageNameLabel]
	if !ok {
		return fmt.Errorf("StageName label %s not found in stageVersion", pkgctrl.StageNameLabel)
	}

	// check if a vectorDeployment exists matching the stage vector
	vectorDeployment, err := r.getOrCreateVectorDeployment(ctx, stageVersion)
	if err != nil {
		return err
	}

	// set vectorDeploymentCreated status
	meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorDeploymentCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.VectorDeploymentCreatedCondition,
		Message:            fmt.Sprintf("Successfully created VectorDeployment %s for stageVersion %s", vectorDeployment.Name, stageVersion.Name),
		ObservedGeneration: stageVersion.Generation,
		LastTransitionTime: metav1.Now(),
	})

	// gate migration on the vectorDeployment being fully ready, so vector-scoped data is distributed before
	// migration tasks run
	if !meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, konfidence.VectorReadyCondition) {
		// wait for vectorDeployment status change notification
		return nil
	}

	// check if a vectorMigration exists matching the stage vector
	vectorMigration, err := r.getOrCreateVectorMigration(ctx, stageVersion)
	if err != nil {
		return err
	}

	// set vectorMigrationCreated status
	meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorMigrationCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.VectorMigrationCreatedCondition,
		Message:            fmt.Sprintf("Successfully created vectorMigration %s for stageVersion %s", vectorMigration.Name, stageVersion.Name),
		ObservedGeneration: stageVersion.Generation,
		LastTransitionTime: metav1.Now(),
	})

	// check if vectorMigration is marked as successful
	if !meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, konfidence.VectorMigrationSucceeded) {
		// wait for vectorMigration status change notification
		return nil
	}

	// set vectorMigrated status
	meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorMigratedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.VectorMigratedCondition,
		Message:            fmt.Sprintf("VectorMigration %s successful for stageVersion %s", vectorMigration.Name, stageVersion.Name),
		ObservedGeneration: stageVersion.Generation,
		LastTransitionTime: metav1.Now(),
	})

	// check if a vectorActivation exists matching the stage vector
	vectorActivation, err := r.getOrCreateVectorActivation(ctx, stageVersion, stageName, vectorDeployment)
	if err != nil {
		return err
	}

	// set vectorActivationCreated status
	meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorActivationCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.VectorActivationCreatedCondition,
		Message:            fmt.Sprintf("Successfully created vectorActivation %s for stageVersion %s", vectorActivation.Name, stageVersion.Name),
		ObservedGeneration: stageVersion.Generation,
		LastTransitionTime: metav1.Now(),
	})

	// set stageVersionReady status
	meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{
		Type:               konfidence.StageVersionReady,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.StageVersionReady,
		Message:            fmt.Sprintf("StageVersion %s reconciled successfully", stageVersion.Name),
		ObservedGeneration: stageVersion.Generation,
		LastTransitionTime: metav1.Now(),
	})

	log.Info("StageVersion reconciled")
	return nil
}

//nolint:dupl // Mirrors child reconciliation in stage_controller.go; keeping explicit resource-specific flow is clearer than a generic helper.
func (r *StageVersionReconciler) getOrCreateVectorDeployment(ctx context.Context, stageVersion *konfidence.StageVersion) (*konfidence.VectorDeployment, error) {
	log := logf.FromContext(ctx)
	vectorDeployment, err := r.constructVectorDeployment(stageVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to construct vectorDeployment from template: %w", err)
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, vectorDeployment, func() error {
		// check if vectorDeployment has stageVersion owner ref
		if err := SetOwnerReference(stageVersion, vectorDeployment, r.Scheme, false); err != nil {
			return fmt.Errorf("unable to check vectorDeployment owner reference: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create or update vectorDeployment: %w", err)
	}
	msg := fmt.Sprintf("VectorDeployment %s for StageVersion %s: %s", vectorDeployment.Name, stageVersion.Name, operationResult)
	r.Recorder.Eventf(stageVersion, nil, corev1.EventTypeNormal, "VectorDeploymentCreated", "VectorDeploymentCreated", msg)
	log.Info(msg)

	return vectorDeployment, nil
}

func (r *StageVersionReconciler) getOrCreateVectorMigration(ctx context.Context, stageVersion *konfidence.StageVersion) (*konfidence.VectorMigration, error) {
	log := logf.FromContext(ctx)
	vectorMigration, err := r.constructVectorMigration(stageVersion)
	if err != nil {
		return nil, fmt.Errorf("unable to construct vectorMigration from template: %w", err)
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, vectorMigration, func() error {
		// check if vectorMigration has stageVersion controller ref
		if err := SetOwnerReference(stageVersion, vectorMigration, r.Scheme, true); err != nil {
			return fmt.Errorf("unable to check vectorMigration owner reference: %w", err)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to create or update vectorMigration: %w", err)
	}
	msg := fmt.Sprintf("Created VectorMigration %s for StageVersion %s", vectorMigration.Name, stageVersion.Name)
	r.Recorder.Eventf(stageVersion, nil, corev1.EventTypeNormal, "VectorMigrationCreated", "VectorMigrationCreated", msg)
	log.V(1).Info(msg)
	return vectorMigration, nil
}

func (r *StageVersionReconciler) getOrCreateVectorActivation(
	ctx context.Context,
	stageVersion *konfidence.StageVersion,
	stageName string,
	vectorDeployment *konfidence.VectorDeployment,
) (*konfidence.VectorActivation, error) {
	log := logf.FromContext(ctx)
	vectorActivation, err := r.constructVectorActivation(stageVersion, stageName, vectorDeployment)
	if err != nil {
		return nil, fmt.Errorf("unable to construct vectorActivation from template: %w", err)
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, vectorActivation, func() error {
		// check if vectorActivation has stageVersion controller ref
		if err := SetOwnerReference(stageVersion, vectorActivation, r.Scheme, true); err != nil {
			return fmt.Errorf("unable to check vectorActivation owner reference: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create or update vectorActivation: %w", err)
	}
	msg := fmt.Sprintf("VectorActivation %s for StageVersion %s: %s", vectorActivation.Name, stageVersion.Name, operationResult)
	r.Recorder.Eventf(stageVersion, nil, corev1.EventTypeNormal, "VectorActivationCreated", "VectorActivationCreated", msg)
	log.V(1).Info(msg)

	return vectorActivation, nil
}

func (r *StageVersionReconciler) constructVectorDeployment(stageVersion *konfidence.StageVersion) (*konfidence.VectorDeployment, error) {
	vectorDeployment := &konfidence.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageVersion.Name,
			Namespace: stageVersion.Namespace,
			Labels:    getVectorDeploymentLabels(stageVersion),
		},
		Spec: konfidence.VectorDeploymentSpec{
			Vector: stageVersion.Spec.Vector,
		},
	}

	// set stageVersion as owner
	if err := controllerutil.SetOwnerReference(stageVersion, vectorDeployment, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set owner reference for vector deployment: %w", err)
	}

	return vectorDeployment, nil
}

func (r *StageVersionReconciler) constructVectorMigration(stageVersion *konfidence.StageVersion) (*konfidence.VectorMigration, error) {
	vectorMigration := &konfidence.VectorMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageVersion.Name,
			Namespace: stageVersion.Namespace,
		},
		Spec: konfidence.VectorMigrationSpec{
			Vector:       stageVersion.Spec.Vector,
			StageVersion: stageVersion.Name,
		},
	}

	// set stageVersion as controller
	if err := controllerutil.SetControllerReference(stageVersion, vectorMigration, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for vector migration: %w", err)
	}
	return vectorMigration, nil
}

func (r *StageVersionReconciler) constructVectorActivation(
	stageVersion *konfidence.StageVersion,
	stageName string,
	vectorDeployment *konfidence.VectorDeployment,
) (*konfidence.VectorActivation, error) {
	vectorActivation := &konfidence.VectorActivation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageVersion.Name,
			Namespace: stageVersion.Namespace,
		},
		Spec: konfidence.VectorActivationSpec{
			Stage:            stageName,
			StageVersion:     stageVersion.Name,
			Vector:           stageVersion.Spec.Vector,
			VectorDeployment: vectorDeployment.Name,
		},
	}

	// set stageVersion as controller
	if err := controllerutil.SetControllerReference(stageVersion, vectorActivation, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for vector activation: %w", err)
	}
	return vectorActivation, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.StageVersion{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&konfidence.VectorMigration{}).
		Owns(&konfidence.VectorActivation{}).
		Watches(
			&konfidence.VectorDeployment{},
			handler.EnqueueRequestsFromMapFunc(reconcileStageVersionOwner),
		).
		Named("stageVersion").
		Complete(r)
}

func reconcileStageVersionOwner(ctx context.Context, obj client.Object) []reconcile.Request {
	// get all stageVersion owners of the watched object
	refs := obj.GetOwnerReferences()
	var stageVersionRefs []metav1.OwnerReference
	for i := range refs {
		if refs[i].Kind == konfidence.StageVersionKind {
			stageVersionRefs = append(stageVersionRefs, refs[i])
		}
	}

	// call reconciliation for each stageVersion owner
	requests := make([]reconcile.Request, 0, len(stageVersionRefs))
	for i := range stageVersionRefs {
		requests = append(requests,
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      stageVersionRefs[i].Name,
					Namespace: obj.GetNamespace(),
				},
			})
	}

	return requests
}

func getVectorDeploymentLabels(stageVersion *konfidence.StageVersion) map[string]string {
	return map[string]string{
		pkgctrl.StageVersionNameLabel: stageVersion.Name,
	}
}
