package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	konfcompref "github.com/konfidence-project/pkg/ocm/compref"
	"github.com/konfidence-project/pkg/ocm/repository"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	"github.com/konfidence-project/gcp-vector-promotion-controller/internal/controller/domain"
)

// VectorPromotionReconciler reconciles a VectorPromotion object.
type VectorPromotionReconciler struct {
	Mgr               mcmanager.Manager
	Scheme            *runtime.Scheme
	OcmClientProvider repository.ClientProvider
	PortProvider      domain.OcmPromotionPortProvider
}

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch

func (r *VectorPromotionReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("cluster", req.ClusterName)
	ctx = logf.IntoContext(ctx, log)
	log.Info("reconciling vector promotion")

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	clusterClient := cluster.GetClient()

	vectorPromotion := &global.VectorPromotion{}
	if err := clusterClient.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Take the snapshot before any modifications for the status patch.
	original := vectorPromotion.DeepCopy()

	if domain.IsRunning(vectorPromotion) { // Promotion was started but promotion status could not be patched, so result is unknown
		if err := setAndPatchPromotionCondition(
			ctx, clusterClient, vectorPromotion, original,
			metav1.ConditionUnknown, global.ReasonPromotionStatusUnknown,
			"Promotion is in unknown state - cannot ensure one-time execution. Aborting reconciliation."); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	config, err := getPromotionConfig(ctx, clusterClient, vectorPromotion)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = fmt.Errorf("promotion configuration %q not found: %w", vectorPromotion.Spec.VectorPromotionConfigRef, err)
			if patchErr := setAndPatchPromotionCondition(
				ctx, clusterClient, vectorPromotion, original, metav1.ConditionFalse,
				global.ReasonPromotionConfigurationNotFound, err.Error()); patchErr != nil {
				return ctrl.Result{}, errors.Join(err, patchErr)
			}
			return ctrl.Result{}, reconcile.TerminalError(err)
		}

		err = fmt.Errorf("failed to fetch promotion configuration %q: %w", vectorPromotion.Spec.VectorPromotionConfigRef, err)
		if patchErr := setAndPatchPromotionCondition(
			ctx, clusterClient, vectorPromotion, original, metav1.ConditionFalse,
			global.ReasonPromotionFailed, err.Error()); patchErr != nil {
			return ctrl.Result{}, errors.Join(err, patchErr)
		}
		return ctrl.Result{}, err
	}

	src, dst, err := parsePromotionParameters(config)
	if err != nil {
		if patchError := setAndPatchPromotionCondition(
			ctx, clusterClient, vectorPromotion, original, metav1.ConditionFalse,
			global.ReasonInvalidPromotionConfiguration, err.Error()); patchError != nil {
			return ctrl.Result{}, errors.Join(err, patchError)
		}
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	ocmClient, err := r.OcmClientProvider.NewClient(ctx, clusterClient, config.GetNamespace(), config.Config)
	if err != nil {
		err = fmt.Errorf("failed to create OCM client: %w", err)
		if patchErr := setAndPatchPromotionCondition(
			ctx, clusterClient, vectorPromotion, original, metav1.ConditionFalse,
			global.ReasonPromotionFailed, err.Error()); patchErr != nil {
			return ctrl.Result{}, errors.Join(err, patchErr)
		}
		return ctrl.Result{}, err
	}

	if err := setAndPatchPromotionCondition(
		ctx, clusterClient, vectorPromotion, original, metav1.ConditionFalse,
		global.ReasonPromotionRunning, "Promotion is currently running"); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.PortProvider.NewOcmPromotionPort(ocmClient).Promote(ctx, *src, *dst); err != nil {
		log.Error(err, "Promotion failed")
		if patchError := setAndPatchPromotionCondition(
			ctx, clusterClient, vectorPromotion, original, metav1.ConditionFalse,
			domain.ClassifyPromotionError(err), err.Error()); patchError != nil {
			return ctrl.Result{}, errors.Join(err, patchError)
		}
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	if err := setAndPatchPromotionCondition(
		ctx, clusterClient, vectorPromotion, original, metav1.ConditionTrue,
		global.ReasonPromotionSucceeded, "Promotion completed successfully"); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func setAndPatchPromotionCondition(
	ctx context.Context,
	clusterClient client.Client,
	vectorPromotion, original *global.VectorPromotion,
	status metav1.ConditionStatus,
	reason, message string) error {
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               global.ConditionTypeSucceeded,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
	if !reflect.DeepEqual(vectorPromotion.Status, original.Status) {
		if err := clusterClient.Status().Patch(ctx, vectorPromotion, client.MergeFrom(original)); err != nil {
			return fmt.Errorf("failed to patch VectorPromotion status: %w", err)
		}
	}
	return nil
}

func parsePromotionParameters(config *global.VectorPromotionConfig) (source, target *compref.Ref, err error) {
	source, err = konfcompref.Parse(config.Spec.Source)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse source reference %q: %w", config.Spec.Source, err)
	}
	target, err = konfcompref.Parse(config.Spec.Target, konfcompref.WithVersionValidation(konfcompref.VersionValidationAliasOnly))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse target reference %q: %w", config.Spec.Target, err)
	}

	if source.Component != target.Component {
		return nil, nil, fmt.Errorf("source and target component names do not match")
	}
	return source, target, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorPromotionReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		For(&global.VectorPromotion{}, mcbuilder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return false },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotion").
		Complete(r)
}
