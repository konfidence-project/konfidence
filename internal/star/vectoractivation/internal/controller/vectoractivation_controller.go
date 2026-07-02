package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/star/vectoractivation/internal/activation"
	leaselock "github.com/konfidence-project/konfidence/internal/star/vectoractivation/internal/lock"
	"github.com/konfidence-project/konfidence/internal/star/vectoractivation/internal/usage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// VectorActivationReconciler reconciles a VectorActivation object
type VectorActivationReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Config       *rest.Config
	ControllerID string
	Recorder     events.EventRecorder
}

const (
	ActivationControllerName = "vector-activation-controller"
)

type ActivationContext struct {
	VectorActivation *star.VectorActivation
	StageVersion     *star.StageVersion
	Stage            *star.Stage
}

// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectoractivations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectoractivations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=activationtaskexecutions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=activationtaskexecutions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=activationtaskregistrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=activationtaskregistrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversionusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversionusages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *VectorActivationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("VectorActivation reconcile started...")

	vectorActivation, stageVersion, stage, err := r.LoadActivationContextData(ctx, req)
	if err != nil || vectorActivation == nil || stageVersion == nil || stage == nil {
		return ctrl.Result{}, fmt.Errorf("could not load activation context data: %w", err)
	}

	if activation.InFinalStatusCondition(vectorActivation) {
		return r.cleanupVectorActivation(ctx, req, vectorActivation, stage)
	}

	acquired, err := leaselock.AcquireResourceLease(
		ctx, r.Client, string(vectorActivation.UID), req.Namespace, r.ControllerID, star.VectorActivationKind, stage,
	)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to acquire lease: %w", err)
	}
	if !acquired {
		log.Info("Lease not acquired, requeuing")
		return ctrl.Result{RequeueAfter: leaselock.DefaultLeaseTTL}, nil
	}
	log.Info("Lease acquired")
	r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "LeaseAcquired", "LeaseAcquired",
		fmt.Sprintf("Lease acquired by controller %s for VectorActivation %s", r.ControllerID, vectorActivation.Name))

	activeStageVersionUsage, err := usage.GetCurrentActiveUsage(ctx, r.Client, stage)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get current active usage: %w", err)
	}

	// check if activation can be skipped
	if activeStageVersionUsage != nil {
		isNewer, err := usage.IsNewerThanCurrentActiveUsage(ctx, r.Client, stageVersion, activeStageVersionUsage)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to compare stage versions: %w", err)
		}
		if !isNewer {
			log.Info("activation belongs to older stage version than currently active one, skipping")
			if err := activation.UpdateVectorActivationStatus(ctx, r.Client, vectorActivation, metav1.Condition{
				Type:               star.ActivationSkipped,
				Status:             metav1.ConditionTrue,
				Reason:             star.ActivationSkipped,
				Message:            "found newer activation",
				ObservedGeneration: vectorActivation.Generation,
				LastTransitionTime: metav1.Now(),
			}); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "ActivationSkipped", "ActivationSkipped",
				fmt.Sprintf("Activation skipped because stage version %s is not newer than currently active stage version %s",
					stageVersion.Name, activeStageVersionUsage.Spec.StageVersionRef))
			return ctrl.Result{}, nil
		}
	}

	activationUsage, err := usage.CreateActivationUsage(ctx, r.Client, stage, vectorActivation)
	if err != nil {
		return ctrl.Result{}, err
	}

	registrationList, err := activation.GetRegistrations(ctx, r.Client, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := activation.UpdateVectorActivationStatus(ctx, r.Client, vectorActivation, metav1.Condition{
		Type:               star.ActivationInProgress,
		Status:             metav1.ConditionTrue,
		Reason:             star.ActivationInProgress,
		Message:            "read in registrations, activation is in progress",
		ObservedGeneration: vectorActivation.Generation,
		LastTransitionTime: metav1.Now(),
	}); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "ActivationInProgress", "ActivationInProgress",
		fmt.Sprintf("VectorActivation %s is in progress", vectorActivation.Name))

	executionsInActivation, err := activation.EnsureExecutionsForRegistrations(ctx, r.Client, req.Namespace, registrationList, vectorActivation)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not create executions: %w", err)
	}
	r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "ExecutionsEnsured", "ExecutionsEnsured",
		fmt.Sprintf("Ensured %d executions for %d registrations", len(executionsInActivation.Items), len(registrationList.Items)))

	allExecutionsSucceeded, err := r.checkExecutionsStatusAndPatchOnFailure(ctx, vectorActivation, executionsInActivation, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if allExecutionsSucceeded {
		r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "ExecutionsSucceeded", "ExecutionsSucceeded",
			fmt.Sprintf("All executions in VectorActivation %s succeeded", vectorActivation.Name))
		log.Info("all executions in activation succeeded")
		if err := usage.CreateOrUpdateActiveUsage(ctx, r.Client, activeStageVersionUsage, stage, stageVersion); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "UsagesUpdated", "UsagesUpdated",
			fmt.Sprintf("Active StageVersionUsage updated to %s", stageVersion.Name))
		if err = usage.DeleteActivationUsage(ctx, r.Client, stage, vectorActivation); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "ActivationUsageDeleted", "ActivationUsageDeleted",
			fmt.Sprintf("Activation StageVersionUsage %s deleted", activationUsage.Name))

		successMessage := fmt.Sprintf("VectorActivation %s reconciled successfully, set status to succeeded", vectorActivation.Name)
		if err := activation.UpdateVectorActivationStatus(ctx, r.Client, vectorActivation, metav1.Condition{
			Type:               star.ActivationSucceeded,
			Status:             metav1.ConditionTrue,
			Reason:             star.ActivationSucceeded,
			Message:            successMessage,
			ObservedGeneration: vectorActivation.Generation,
			LastTransitionTime: metav1.Now(),
		}); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "ActivationSucceeded", "ActivationSucceeded", successMessage)
		log.Info(successMessage)
	}

	return ctrl.Result{}, nil
}

func (r *VectorActivationReconciler) checkExecutionsStatusAndPatchOnFailure(
	ctx context.Context,
	vectorActivation *star.VectorActivation,
	executionsInActivation *star.ActivationTaskExecutionList,
	log logr.Logger,
) (bool, error) {
	allExecutionsSucceeded := true

	for _, exec := range executionsInActivation.Items {
		if meta.IsStatusConditionTrue(exec.Status.Conditions, star.ActivationTaskExecutionFailed) {
			msg := fmt.Sprintf("ActivationTaskExecution failed: %s", exec.Name)
			log.Info(msg)
			if err := activation.UpdateVectorActivationStatus(ctx, r.Client, vectorActivation, metav1.Condition{
				Type:               star.ActivationFailed,
				Status:             metav1.ConditionTrue,
				Reason:             star.ActivationTaskExecutionFailed,
				Message:            msg,
				ObservedGeneration: vectorActivation.Generation,
				LastTransitionTime: metav1.Now(),
			}); err != nil {
				return false, err
			}
			r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "ActivationFailed", "ActivationFailed",
				fmt.Sprintf("VectorActivation %s failed because execution %s failed", vectorActivation.Name, exec.Name))
			allExecutionsSucceeded = false
		}
		if !meta.IsStatusConditionTrue(exec.Status.Conditions, star.ActivationTaskExecutionSucceeded) {
			allExecutionsSucceeded = false
			break
		}
	}
	return allExecutionsSucceeded, nil
}

func (r *VectorActivationReconciler) LoadActivationContextData(
	ctx context.Context, req ctrl.Request,
) (*star.VectorActivation, *star.StageVersion, *star.Stage, error) {
	vectorActivation := &star.VectorActivation{}
	if err := r.Get(ctx, req.NamespacedName, vectorActivation); err != nil {
		return nil, nil, nil, client.IgnoreNotFound(err)
	}

	stageVersion := &star.StageVersion{}
	if err := r.Get(ctx, types.NamespacedName{Name: vectorActivation.Spec.StageVersion, Namespace: req.Namespace}, stageVersion); err != nil {
		return nil, nil, nil, fmt.Errorf("could not get stage version: %w", err)
	}

	stage := &star.Stage{}
	if err := r.Get(ctx, types.NamespacedName{Name: vectorActivation.Spec.Stage, Namespace: req.Namespace}, stage); err != nil {
		return nil, nil, nil, fmt.Errorf("could not get stage: %w", err)
	}

	return vectorActivation, stageVersion, stage, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorActivationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ControllerID = os.Getenv("POD_NAME")
	if r.ControllerID == "" {
		r.ControllerID = ActivationControllerName
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&star.VectorActivation{}).
		Named("vectoractivation").
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Owns(&star.ActivationTaskExecution{},
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc:  func(e event.UpdateEvent) bool { return true },
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				CreateFunc:  func(e event.CreateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			})).
		Complete(r)
}

func (r *VectorActivationReconciler) cleanupVectorActivation(
	ctx context.Context,
	req ctrl.Request,
	vectorActivation *star.VectorActivation,
	stage *star.Stage,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	if err := usage.DeleteActivationUsage(ctx, r.Client, stage, vectorActivation); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete activation usage: %w", err)
	}
	log.Info("release lease for vectorActivation")
	r.Recorder.Eventf(vectorActivation, nil, corev1.EventTypeNormal, "LeaseReleased", "LeaseReleased",
		fmt.Sprintf("Lease released by controller %s for VectorActivation %s", r.ControllerID, vectorActivation.Name))
	if err := leaselock.ReleaseResourceLease(
		ctx, r.Client, string(vectorActivation.UID), req.Namespace, r.ControllerID, star.VectorActivationKind, stage,
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to release lease: %w", err)
	}
	return ctrl.Result{}, nil
}
