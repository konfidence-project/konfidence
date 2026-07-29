package controller

import (
	"context"
	"fmt"
	"reflect"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	"github.com/konfidence-project/konfidence/pkg/hash"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const StageControllerName = "stage-controller"

// StageReconciler reconciles a Stage object
type StageReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stageversions/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile

func (r *StageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stage started...")

	// get stage
	stage := &konfidence.Stage{}
	if err := r.Get(ctx, req.NamespacedName, stage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStage := stage.DeepCopy()
	err := r.reconcileStage(ctx, req, stage)

	return ctrl.Result{}, pkgctrl.PatchStatusIfChanged(
		ctx,
		r.Client,
		stage,
		originalStage,
		stage.Status,
		originalStage.Status,
		"unable to update stage status",
		err,
		"an error occurred while reconciling stage",
	)

}

func (r *StageReconciler) reconcileStage(ctx context.Context, req ctrl.Request, stage *konfidence.Stage) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stage")
	meta.SetStatusCondition(&stage.Status.Conditions, metav1.Condition{
		Type:               konfidence.StageReady,
		Status:             metav1.ConditionFalse,
		Reason:             konfidence.StageReady,
		Message:            "",
		ObservedGeneration: stage.Generation,
		LastTransitionTime: metav1.Now(),
	})

	_, err := r.getOrCreateTargetStageVersionUsage(ctx, req, stage)
	if err != nil {
		return err
	}

	_, err = r.getOrCreateStageVersion(ctx, stage)
	if err != nil {
		return err
	}

	meta.SetStatusCondition(&stage.Status.Conditions, metav1.Condition{
		Type:               konfidence.StageReady,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.StageReady,
		Message:            fmt.Sprintf("Successfully reconciled Stage %s", stage.Name),
		ObservedGeneration: stage.Generation,
		LastTransitionTime: metav1.Now(),
	})
	log.Info("Stage reconciled")
	return nil
}

func (r *StageReconciler) getOrCreateTargetStageVersionUsage(
	ctx context.Context,
	req ctrl.Request,
	stage *konfidence.Stage,
) (*konfidence.StageVersionUsage, error) {
	log := logf.FromContext(ctx)

	stageVersionUsages := &konfidence.StageVersionUsageList{}
	if err := r.List(ctx, stageVersionUsages, client.InNamespace(req.Namespace), client.MatchingLabels(getTargetStageVersionUsageLabels(stage))); err != nil {
		return nil, fmt.Errorf("unable to list current target stageVersionUsages: %w", err)
	}

	if len(stageVersionUsages.Items) == 0 {
		log.Info("No matching target stageVersionUsage found. Creating a new one...")

		// create new stageVersionUsage
		stageVersionUsage, err := r.constructStageVersionUsage(stage)
		if err != nil {
			return nil, fmt.Errorf("unable to construct target stageVersionUsage from template: %w", err)
		}

		if err := r.Create(ctx, stageVersionUsage); err != nil {
			return nil, fmt.Errorf("unable to create target stageVersionUsage: %w", err)
		}

		msg := fmt.Sprintf("Created target StageVersionUsage %s", stageVersionUsage.Name)
		r.Recorder.Eventf(stage, nil, corev1.EventTypeNormal, "StageVersionUsageCreated", "StageVersionUsageCreated", msg)

		log.Info(msg)
		return stageVersionUsage, nil
	}

	// found one or more matching target stageVersionUsages
	// we just use the first one found
	stageVersionUsage := &stageVersionUsages.Items[0]
	log.Info("Found existing target stageVersionUsage", "stageVersionUsage", stageVersionUsage)

	// update the target usage with the current spec
	if err := r.updateTargetStageVersionUsage(ctx, stageVersionUsage, stage); err != nil {
		return nil, fmt.Errorf("unable to update target stageVersionUsage specs: %w", err)
	}

	if len(stageVersionUsages.Items) > 1 {
		log.Info("Deleting obsolete target stageVersionUsages")

		// delete all other (potentially manually) created target stageVersionUsages
		for _, stageVersionUsage := range stageVersionUsages.Items[1:] {
			if err := r.Delete(ctx, &stageVersionUsage); err != nil {
				return nil, fmt.Errorf("unable to delete obsolete target stageVersionUsage: %w", err)
			}
			msg := fmt.Sprintf("Deleted obsolete target StageVersionUsage %s", stageVersionUsage.Name)
			r.Recorder.Eventf(stage, nil, corev1.EventTypeNormal, "StageVersionUsageDeleted", "StageVersionUsageDeleted", msg)
			log.Info(msg)
		}
	}

	return stageVersionUsage, nil
}

//nolint:dupl // Mirrors child reconciliation in stage_version_controller.go; keeping explicit resource-specific flow is clearer than a generic helper.
func (r *StageReconciler) getOrCreateStageVersion(ctx context.Context, stage *konfidence.Stage) (*konfidence.StageVersion, error) {
	log := logf.FromContext(ctx)
	stageVersion, err := r.constructStageVersion(stage)
	if err != nil {
		return nil, fmt.Errorf("unable to construct stageVersion from template: %w", err)
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, stageVersion, func() error {
		if err := SetOwnerReference(stage, stageVersion, r.Scheme, true); err != nil {
			return fmt.Errorf("unable to check stageVersion owner reference: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create or update stageVersion: %w", err)
	}

	msg := fmt.Sprintf("StageVersion %s for Stage %s: %s", stageVersion.Name, stage.Name, operationResult)
	r.Recorder.Eventf(stage, nil, corev1.EventTypeNormal, "StageVersionReconciled", "StageVersionReconciled", msg)
	log.Info(msg)

	return stageVersion, nil
}

func (r *StageReconciler) constructStageVersion(stage *konfidence.Stage) (*konfidence.StageVersion, error) {
	name := getStageVersionName(stage)

	stageVersionLabels := getStageVersionLabels(stage)

	stageVersion := &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stage.Namespace,
			Labels:    stageVersionLabels,
		},
		Spec: konfidence.StageVersionSpec{
			Vector:          stage.Spec.Vector,
			StageGeneration: stage.Generation,
			StageRef: &konfidence.StageReference{
				Name: stage.Name,
			},
		},
	}

	if err := ctrl.SetControllerReference(stage, stageVersion, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for stageVersion: %w", err)
	}

	return stageVersion, nil
}

func (r *StageReconciler) constructStageVersionUsage(stage *konfidence.Stage) (*konfidence.StageVersionUsage, error) {
	name := fmt.Sprintf("%s-target-%s", stage.Name, rand.String(8))
	stageVersionLabels := getStageVersionLabels(stage)

	stageVersionUsage := &konfidence.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stage.Namespace,
			Labels:    getTargetStageVersionUsageLabels(stage),
		},
		Spec: konfidence.StageVersionUsageSpec{
			Reason: StageVersionUsageTargetType,
			StageVersionSelector: &metav1.LabelSelector{
				MatchLabels: stageVersionLabels,
			},
		},
	}

	if err := controllerutil.SetControllerReference(stage, stageVersionUsage, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for stageVersionUsage: %w", err)
	}

	return stageVersionUsage, nil
}

func (r *StageReconciler) updateTargetStageVersionUsage(ctx context.Context, stageVersionUsage *konfidence.StageVersionUsage, stage *konfidence.Stage) error {
	stageVersionLabels := getStageVersionLabels(stage)

	originalStageVersionUsage := stageVersionUsage.DeepCopy()
	patch := client.MergeFrom(originalStageVersionUsage)

	stageVersionUsage.Labels = getTargetStageVersionUsageLabels(stage)
	stageVersionUsage.Spec.Reason = StageVersionUsageTargetType
	stageVersionUsage.Spec.StageVersionRef = nil
	stageVersionUsage.Spec.StageVersionSelector = &metav1.LabelSelector{
		MatchLabels: stageVersionLabels,
	}

	// remove all owner references
	stageVersionUsage.OwnerReferences = []metav1.OwnerReference{}

	// and set stage controller reference
	if err := controllerutil.SetControllerReference(stage, stageVersionUsage, r.Scheme); err != nil {
		return fmt.Errorf("unable to set controller reference for stageVersionUsage: %w", err)
	}

	if !reflect.DeepEqual(stageVersionUsage, originalStageVersionUsage) {
		if err := r.Patch(ctx, stageVersionUsage, patch); err != nil {
			return fmt.Errorf("unable to patch target stageVersionUsage: %w", err)
		}
	}

	return nil
}

func getStageVersionLabels(stage *konfidence.Stage) map[string]string {
	// use vector hash as vector ref for now
	digest := hash.Fnv(stage.Spec.Vector, 25)

	return map[string]string{
		pkgctrl.StageNameLabel:       stage.Name,
		pkgctrl.VectorReferenceLabel: digest,
	}
}

func getTargetStageVersionUsageLabels(stage *konfidence.Stage) map[string]string {
	return map[string]string{
		pkgctrl.StageVersionUsageTarget: stage.Name,
	}
}

func getStageVersionName(stage *konfidence.Stage) string {
	content := fmt.Sprintf("%s-%s-%d", stage.Name, stage.Spec.Vector, stage.Generation)
	digest := hash.Fnv(content, 13)
	return fmt.Sprintf("%s-%s", stage.Name, digest)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	noUpdatePredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},

		// Allow create events
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},

		// Allow delete events
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},

		// Allow generic events (e.g., external triggers)
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.Stage{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&konfidence.StageVersion{}, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, noUpdatePredicate))).
		Owns(&konfidence.StageVersionUsage{}, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, noUpdatePredicate))).
		Named("stage").
		Complete(r)
}
