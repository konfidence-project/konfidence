package stage

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// StageVersionUsageReconciler reconciles a StageVersionUsage object
type StageVersionUsageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversionusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversionusages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversions/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
//
//nolint:dupl // TODO(konfidence-project#689): factor shared reconcile/status-patch boilerplate.
func (r *StageVersionUsageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stageVersionUsage started...")

	// get stageVersionUsage
	stageVersionUsage := &star.StageVersionUsage{}
	if err := r.Get(ctx, req.NamespacedName, stageVersionUsage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStageVersionUsage := stageVersionUsage.DeepCopy()
	patch := client.MergeFrom(originalStageVersionUsage)
	err := r.reconcileStageVersionUsage(ctx, req, stageVersionUsage)

	if !reflect.DeepEqual(stageVersionUsage.Status, originalStageVersionUsage.Status) {
		if patchError := r.Client.Status().Patch(ctx, stageVersionUsage, patch); patchError != nil {
			patchErrorMessage := "unable to update stageVersionUsage status"

			if err != nil {
				reconcileError := fmt.Errorf("an error occurred while reconciling stageVersionUsage: %w", err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}

	return ctrl.Result{}, err
}

func (r *StageVersionUsageReconciler) reconcileStageVersionUsage(ctx context.Context, req ctrl.Request, stageVersionUsage *star.StageVersionUsage) error {
	resolvedStageVersions, err := r.resolveStageVersions(ctx, req, stageVersionUsage)
	if err != nil {
		return err
	}

	if resolvedStageVersions == nil {
		return nil
	}

	meta.RemoveStatusCondition(&stageVersionUsage.Status.Conditions, star.StageVersionNotFound)

	allStageVersionsReady := true
	for _, stageVersion := range resolvedStageVersions {
		if !meta.IsStatusConditionTrue(stageVersion.Status.Conditions, star.StageVersionReady) {
			allStageVersionsReady = false
			break
		}
	}

	if allStageVersionsReady {
		meta.SetStatusCondition(&stageVersionUsage.Status.Conditions, metav1.Condition{
			Type:               star.StageVersionUsageReady,
			Status:             metav1.ConditionTrue,
			Reason:             star.StageVersionUsageReady,
			Message:            "Referenced StageVersion(s) are rolled out and ready for traffic",
			ObservedGeneration: stageVersionUsage.Generation,
			LastTransitionTime: metav1.Now(),
		})
	} else {
		meta.SetStatusCondition(&stageVersionUsage.Status.Conditions, metav1.Condition{
			Type:               star.StageVersionUsageReady,
			Status:             metav1.ConditionFalse,
			Reason:             star.StageVersionUsageReady,
			Message:            "Referenced StageVersion(s) are not ready",
			ObservedGeneration: stageVersionUsage.Generation,
			LastTransitionTime: metav1.Now(),
		})
	}

	// update status with current resolved stageVersion names
	stageVersionNames := make([]string, 0, len(resolvedStageVersions))
	for _, stageVersion := range resolvedStageVersions {
		stageVersionNames = append(stageVersionNames, stageVersion.Name)
	}

	sort.Strings(stageVersionNames)
	stageVersionUsage.Status.ResolvedStageVersions = stageVersionNames
	return nil
}

func (r *StageVersionUsageReconciler) resolveStageVersions(
	ctx context.Context, req ctrl.Request, stageVersionUsage *star.StageVersionUsage,
) ([]star.StageVersion, error) {
	log := logf.FromContext(ctx)
	notFoundCondition := metav1.Condition{
		Type:               star.StageVersionNotFound,
		Status:             metav1.ConditionTrue,
		Reason:             star.StageVersionNotFound,
		Message:            "Referenced StageVersion(s) not found",
		ObservedGeneration: stageVersionUsage.Generation,
		LastTransitionTime: metav1.Now(),
	}

	if stageVersionUsage.Spec.StageVersionRef != nil {
		stageVersion := &star.StageVersion{}
		err := r.Get(ctx, types.NamespacedName{Name: stageVersionUsage.Spec.StageVersionRef.Name, Namespace: req.Namespace}, stageVersion)
		if err != nil && errors.IsNotFound(err) {
			meta.SetStatusCondition(&stageVersionUsage.Status.Conditions, notFoundCondition)
			log.Info(fmt.Sprintf("referenced stageVersion %s does not exist", stageVersionUsage.Spec.StageVersionRef.Name))
			return nil, nil
		}

		if err != nil {
			return nil, fmt.Errorf("unable to check referenced stageVersion: %w", err)
		}
		return []star.StageVersion{
			*stageVersion,
		}, nil
	} else {
		// get all stageVersions that match the stageVersionUsage selector
		labelMatcher := client.MatchingLabels{}
		for key, value := range stageVersionUsage.Spec.StageVersionSelector.MatchLabels {
			labelMatcher[key] = value
		}

		stageVersions := &star.StageVersionList{}
		if err := r.List(ctx, stageVersions, client.InNamespace(req.Namespace), labelMatcher); err != nil {
			return nil, fmt.Errorf("unable to list stageVersions: %w", err)
		}

		if len(stageVersions.Items) == 0 {
			meta.SetStatusCondition(&stageVersionUsage.Status.Conditions, notFoundCondition)
			log.Info(fmt.Sprintf("referenced stageVersion(s) do not exist %v", stageVersionUsage.Spec.StageVersionSelector.MatchLabels))
			return nil, nil
		}

		return stageVersions.Items, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageVersionUsageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&star.StageVersionUsage{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&star.StageVersion{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileStageVersionUsages),
		).
		Named("stageVersionUsage").
		Complete(r)
}

func (r *StageVersionUsageReconciler) reconcileStageVersionUsages(ctx context.Context, obj client.Object) []reconcile.Request {
	// get all stageVersionUsages
	// TODO this is very inefficient and should be changed in a later version
	stageVersionUsages := &star.StageVersionUsageList{}
	err := r.List(ctx, stageVersionUsages, client.InNamespace(obj.GetNamespace()))
	if err != nil || len(stageVersionUsages.Items) == 0 {
		return []reconcile.Request{}
	}

	// call reconciliation for each stageVersionUsage
	requests := make([]reconcile.Request, 0, len(stageVersionUsages.Items))
	for i := range stageVersionUsages.Items {
		requests = append(requests,
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      stageVersionUsages.Items[i].Name,
					Namespace: obj.GetNamespace(),
				},
			})
	}

	return requests
}
