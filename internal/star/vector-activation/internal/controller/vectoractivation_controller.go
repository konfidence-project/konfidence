/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"github.com/konfidence-project/landscape-vector-activation-controller/internal/activation"
	leaseLock "github.com/konfidence-project/landscape-vector-activation-controller/internal/lock"
	"github.com/konfidence-project/landscape-vector-activation-controller/internal/usage"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
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
	Recorder     record.EventRecorder
}

const (
	ActivationControllerName = "vectoractivation-controller"
)

type ActivationContext struct {
	VectorActivation *landscape.VectorActivation
	StageVersion     *landscape.StageVersion
	Stage            *common.Stage
}

// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=activationtaskexecutions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=activationtaskexecutions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=activationtaskregistrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=activationtaskregistrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversionusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversionusages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases/status,verbs=get;update;patch

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

	acquired, err := leaseLock.AcquireResourceLease(ctx, r.Client, string(vectorActivation.UID), req.Namespace, r.ControllerID, landscape.VectorActivationKind, stage)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to acquire lease: %w", err)
	}
	if !acquired {
		log.Info("Lease not acquired, requeuing")
		return ctrl.Result{RequeueAfter: leaseLock.DefaultLeaseTTL}, nil
	}
	log.Info("Lease acquired")
	r.Recorder.Event(vectorActivation, "Normal", "LeaseAcquired", fmt.Sprintf("Lease acquired by controller %s for VectorActivation %s", r.ControllerID, vectorActivation.Name))

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
			if err := activation.UpdateVectorActivationStatus(ctx, r.Client, vectorActivation, metav1.Condition{Type: landscape.ActivationSkipped, Status: metav1.ConditionTrue, Reason: landscape.ActivationSkipped, Message: "found newer activation"}); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Event(vectorActivation, "Normal", "ActivationSkipped", fmt.Sprintf("Activation skipped because stage version %s is not newer than currently active stage version %s", stageVersion.Name, activeStageVersionUsage.Spec.StageVersionRef))
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

	if err := activation.UpdateVectorActivationStatus(ctx, r.Client, vectorActivation, metav1.Condition{Type: landscape.ActivationInProgress, Status: metav1.ConditionTrue, Reason: landscape.ActivationInProgress, Message: "read in registrations, activation is in progress"}); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(vectorActivation, "Normal", "ActivationInProgress", fmt.Sprintf("VectorActivation %s is in progress", vectorActivation.Name))

	executionsInActivation, err := activation.EnsureExecutionsForRegistrations(ctx, r.Client, req.Namespace, registrationList, vectorActivation)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not create executions: %w", err)
	}
	r.Recorder.Event(vectorActivation, "Normal", "ExecutionsEnsured", fmt.Sprintf("Ensured %d executions for %d registrations", len(executionsInActivation.Items), len(registrationList.Items)))

	allExecutionsSucceeded, err := r.checkExecutionsStatusAndPatchOnFailure(ctx, vectorActivation, executionsInActivation, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if allExecutionsSucceeded {
		r.Recorder.Event(vectorActivation, "Normal", "ExecutionsSucceeded", fmt.Sprintf("All executions in VectorActivation %s succeeded", vectorActivation.Name))
		log.Info("all executions in activation succeeded")
		if err := usage.CreateOrUpdateActiveUsage(ctx, r.Client, activeStageVersionUsage, stage, stageVersion); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(vectorActivation, "Normal", "UsagesUpdated", fmt.Sprintf("Active StageVersionUsage updated to %s", stageVersion.Name))
		if err = usage.DeleteActivationUsage(ctx, r.Client, activationUsage); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(vectorActivation, "Normal", "ActivationUsageDeleted", fmt.Sprintf("Activation StageVersionUsage %s deleted", activationUsage.Name))

		successMessage := fmt.Sprintf("VectorActivation %s reconciled successfully, set status to succeeded", vectorActivation.Name)
		if err := activation.UpdateVectorActivationStatus(ctx, r.Client, vectorActivation, metav1.Condition{Type: landscape.ActivationSucceeded, Status: metav1.ConditionTrue, Reason: landscape.ActivationSucceeded, Message: successMessage}); err != nil {
			return ctrl.Result{}, err
		}

		r.Recorder.Event(vectorActivation, "Normal", "ActivationSucceeded", successMessage)
		log.Info(successMessage)
	}

	return ctrl.Result{}, nil
}

func (r *VectorActivationReconciler) checkExecutionsStatusAndPatchOnFailure(ctx context.Context, vectorActivation *landscape.VectorActivation, executionsInActivation *landscape.ActivationTaskExecutionList, log logr.Logger) (bool, error) {
	allExecutionsSucceeded := true

	for _, exec := range executionsInActivation.Items {
		if meta.IsStatusConditionTrue(exec.Status.Conditions, landscape.ActivationTaskExecutionFailed) {
			msg := fmt.Sprintf("ActivationTaskExecution failed: %s", exec.Name)
			log.Info(msg)
			if err := activation.UpdateVectorActivationStatus(ctx, r.Client, vectorActivation, metav1.Condition{Type: landscape.ActivationFailed, Status: metav1.ConditionTrue, Reason: landscape.ActivationTaskExecutionFailed, Message: msg}); err != nil {
				return false, err
			}
			r.Recorder.Event(vectorActivation, "Normal", "ActivationFailed", fmt.Sprintf("VectorActivation %s failed because execution %s failed", vectorActivation.Name, exec.Name))
			allExecutionsSucceeded = false
		}
		if !meta.IsStatusConditionTrue(exec.Status.Conditions, landscape.ActivationTaskExecutionSucceeded) {
			allExecutionsSucceeded = false
			break
		}
	}
	return allExecutionsSucceeded, nil
}

func (r *VectorActivationReconciler) LoadActivationContextData(ctx context.Context, req ctrl.Request) (*landscape.VectorActivation, *landscape.StageVersion, *common.Stage, error) {
	vectorActivation := &landscape.VectorActivation{}
	if err := r.Get(ctx, req.NamespacedName, vectorActivation); err != nil {
		return nil, nil, nil, client.IgnoreNotFound(err)
	}

	stageVersion := &landscape.StageVersion{}
	if err := r.Get(ctx, types.NamespacedName{Name: vectorActivation.Spec.StageVersion, Namespace: req.Namespace}, stageVersion); err != nil {
		return nil, nil, nil, fmt.Errorf("could not get stage version: %w", err)
	}

	stage := &common.Stage{}
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
	r.Recorder = mgr.GetEventRecorderFor(ActivationControllerName)
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorActivation{}).
		Named("vectoractivation").
		WithEventFilter(predicate.Funcs{
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Owns(&landscape.ActivationTaskExecution{},
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc:  func(e event.UpdateEvent) bool { return true },
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				CreateFunc:  func(e event.CreateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			})).
		Complete(r)
}

func (r *VectorActivationReconciler) cleanupVectorActivation(ctx context.Context, req ctrl.Request, vectorActivation *landscape.VectorActivation, stage *common.Stage) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("release lease for vectorActivation")
	r.Recorder.Event(vectorActivation, "Normal", "LeaseReleased", fmt.Sprintf("Lease released by controller %s for VectorActivation %s", r.ControllerID, vectorActivation.Name))
	if err := leaseLock.ReleaseResourceLease(ctx, r.Client, string(vectorActivation.UID), req.Namespace, r.ControllerID, landscape.VectorActivationKind, stage); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to release lease: %w", err)
	}
	return ctrl.Result{}, nil
}
