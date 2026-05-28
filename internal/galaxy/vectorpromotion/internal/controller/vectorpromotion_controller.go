package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	konfcompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
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

	"github.com/konfidence-project/konfidence/internal/galaxy/vectorpromotion/internal/promotion"
)

const (
	VectorPromotionControllerName = "galaxy-vector-promotion-controller"

	EventActionUnknownPromotionStatus = "ReconcileRunningPromotion"
	EventActionStatusPatch            = "StatusPatch"
	EventActionReconciling            = "Reconciling"
)

// VectorPromotionReconciler reconciles a VectorPromotion object.
type VectorPromotionReconciler struct {
	Mgr               mcmanager.Manager
	Scheme            *runtime.Scheme
	OcmClientProvider repository.ClientProvider
	PortProvider      promotion.OcmPortProvider
}

// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectorpromotions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch

func (r *VectorPromotionReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("cluster", req.ClusterName)
	ctx = logf.IntoContext(ctx, log)
	log.Info("reconciling vector promotion")

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}
	clusterClient := cluster.GetClient()
	recorder := cluster.GetEventRecorder(VectorPromotionControllerName)

	vectorPromotion := &galaxy.VectorPromotion{}
	if err := clusterClient.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Take the snapshot before any modifications for the status patch.
	original := vectorPromotion.DeepCopy()

	if promotion.IsRunning(vectorPromotion) { // Promotion was started but promotion status could not be patched, so result is unknown
		logStr := `Promotion result is unknown probably because the controller failed to patch the promotion ` +
			`status after starting the promotion. Aborting reconciliation.`
		log.Info(logStr)
		recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "EncounteredRunningPromotion", EventActionUnknownPromotionStatus,
			fmt.Sprintf("%s Please check previous events for details.", logStr))
		if err := setAndPatchPromotionCondition(
			ctx, log, clusterClient, recorder, vectorPromotion, original,
			metav1.ConditionUnknown, galaxy.ReasonPromotionStatusUnknown, logStr); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	config, err := getPromotionConfig(ctx, clusterClient, vectorPromotion)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, fmt.Sprintf("promotion configuration %q not found", vectorPromotion.Spec.VectorPromotionConfigRef))
			err = fmt.Errorf("promotion configuration %q not found: %w", vectorPromotion.Spec.VectorPromotionConfigRef, err)
			if patchErr := setAndPatchPromotionCondition(
				ctx, log, clusterClient, recorder, vectorPromotion, original, metav1.ConditionFalse,
				galaxy.ReasonPromotionConfigurationNotFound, err.Error()); patchErr != nil {
				return ctrl.Result{}, errors.Join(err, patchErr)
			}
			return ctrl.Result{}, reconcile.TerminalError(err)
		}

		log.Error(err, fmt.Sprintf("failed to fetch promotion configuration %q", vectorPromotion.Spec.VectorPromotionConfigRef))
		err = fmt.Errorf("failed to fetch promotion configuration %q: %w", vectorPromotion.Spec.VectorPromotionConfigRef, err)
		if patchErr := setAndPatchPromotionCondition(
			ctx, log, clusterClient, recorder, vectorPromotion, original, metav1.ConditionFalse,
			galaxy.ReasonPromotionFailed, err.Error()); patchErr != nil {
			return ctrl.Result{}, errors.Join(err, patchErr)
		}
		return ctrl.Result{}, err
	}

	src, dst, err := parsePromotionParameters(config)
	if err != nil {
		log.Error(err, "failed to parse promotion parameters")
		if patchError := setAndPatchPromotionCondition(
			ctx, log, clusterClient, recorder, vectorPromotion, original, metav1.ConditionFalse,
			galaxy.ReasonInvalidPromotionConfiguration, err.Error()); patchError != nil {
			return ctrl.Result{}, errors.Join(err, patchError)
		}
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	ocmClient, err := r.OcmClientProvider.NewClient(ctx, clusterClient, config.GetNamespace(), config.Config)
	if err != nil {
		log.Error(err, "failed to create OCM client")
		err = fmt.Errorf("failed to create OCM client: %w", err)
		if patchErr := setAndPatchPromotionCondition(
			ctx, log, clusterClient, recorder, vectorPromotion, original, metav1.ConditionFalse,
			galaxy.ReasonPromotionFailed, err.Error()); patchErr != nil {
			return ctrl.Result{}, errors.Join(err, patchErr)
		}
		return ctrl.Result{}, err
	}

	msgStr := "starting promotion"
	log.Info(msgStr)
	recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionStarting", EventActionReconciling, msgStr)
	if err := setAndPatchPromotionCondition(
		ctx, log, clusterClient, recorder, vectorPromotion, original, metav1.ConditionFalse,
		galaxy.ReasonPromotionRunning, "Promotion is currently running"); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.PortProvider.NewOcmPromotionPort(ocmClient).Promote(ctx, *src, *dst); err != nil {
		log.Error(err, "Promotion failed")
		if patchError := setAndPatchPromotionCondition(
			ctx, log, clusterClient, recorder, vectorPromotion, original, metav1.ConditionFalse,
			promotion.ClassifyPromotionError(err), err.Error()); patchError != nil {
			return ctrl.Result{}, errors.Join(err, patchError)
		}
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	msgStr = "promotion succeeded"
	log.Info(msgStr)
	recorder.Eventf(vectorPromotion, nil, corev1.EventTypeNormal, "PromotionSuccessful", EventActionReconciling, msgStr)
	if err := setAndPatchPromotionCondition(
		ctx, log, clusterClient, recorder, vectorPromotion, original, metav1.ConditionTrue,
		galaxy.ReasonPromotionSucceeded, "Promotion completed successfully"); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func setAndPatchPromotionCondition(
	ctx context.Context,
	log logr.Logger,
	clusterClient client.Client,
	recorder events.EventRecorder,
	vectorPromotion, original *galaxy.VectorPromotion,
	status metav1.ConditionStatus,
	reason, message string) error {
	meta.SetStatusCondition(&vectorPromotion.Status.Conditions, metav1.Condition{
		Type:               galaxy.ConditionTypeSucceeded,
		Status:             status,
		ObservedGeneration: vectorPromotion.Generation,
		Reason:             reason,
		Message:            message,
	})
	if !reflect.DeepEqual(vectorPromotion.Status, original.Status) {
		if err := clusterClient.Status().Patch(ctx, vectorPromotion, client.MergeFrom(original)); err != nil {
			log.Error(err, fmt.Sprintf("failed to patch promotion status of promotion %q in namespace %q",
				vectorPromotion.Name, vectorPromotion.Namespace))
			recorder.Eventf(vectorPromotion, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch,
				fmt.Sprintf("wanted to set condition of type %q to status %q with reason %q and message %q "+
					"but failed with error: %s", galaxy.ConditionTypeSucceeded, status, reason, message, err.Error()))
			return fmt.Errorf("failed to patch VectorPromotion status: %w", err)
		}
	}
	return nil
}

func parsePromotionParameters(config *galaxy.VectorPromotionConfig) (source, target *compref.Ref, err error) {
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
		For(&galaxy.VectorPromotion{}, mcbuilder.WithPredicates(predicate.Funcs{
			CreateFunc:  func(e event.CreateEvent) bool { return true },
			UpdateFunc:  func(e event.UpdateEvent) bool { return false },
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		})).
		Named("vectorPromotion").
		Complete(r)
}
