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
}

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
// +kubebuilder:rbac:groups=coordination,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination,resources=leases/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *VectorActivationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("VectorActivation reconcile started...")

	// TODO: turn the stageVersion in the VectorActivationSpec into a StageVersionRef
	vectorActivation, stageVersion, stage, err := r.LoadActivationContextData(ctx, req)
	if err != nil || vectorActivation == nil || stageVersion == nil || stage == nil {
		return ctrl.Result{}, fmt.Errorf("could not load activation context data: %w", err)
	}

	if activation.InFinalStatusCondition(vectorActivation) {
		return r.cleanupVectorActivation(ctx, req, vectorActivation, stage.Name)
	}

	acquired, err := AcquireLease(ctx, r.Client, r.ControllerID, req.Namespace, vectorActivation, stage)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to acquire lease: %w", err)
	}
	if !acquired {
		log.Info("Lease not acquired, requeuing")
		return ctrl.Result{RequeueAfter: leaseLock.DefaultLeaseTTL}, nil
	}
	log.Info("Lease acquired", "acquired", acquired)

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
			if err := activation.PatchVectorActivationStatus(ctx, r.Client, req.NamespacedName, metav1.Condition{Type: landscape.ActivationSkipped, Status: metav1.ConditionTrue, Reason: landscape.ActivationSkipped, Message: "found newer activation"}); err != nil {
				return ctrl.Result{}, err
			}
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

	if err := activation.PatchVectorActivationStatus(ctx, r.Client, req.NamespacedName, metav1.Condition{Type: landscape.ActivationInProgress, Status: metav1.ConditionTrue, Reason: landscape.ActivationInProgress, Message: "read in registrations, activation is in progress"}); err != nil {
		return ctrl.Result{}, err
	}

	err = activation.EnsureExecutionsForRegistrations(ctx, r.Client, req.Namespace, registrationList, vectorActivation)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not create executions: %w", err)
	}

	executionsInActivation, err := activation.ListExecutions(ctx, r.Client, req.Namespace, vectorActivation)
	if err != nil {
		return ctrl.Result{}, err
	}

	allExecutionsSucceeded, err := r.checkExecutionsStatusAndPatchOnFailure(ctx, req, executionsInActivation, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	if allExecutionsSucceeded {
		log.Info("all executions in activation succeeded")
		if err := usage.CreateOrUpdateActiveUsage(ctx, r.Client, activeStageVersionUsage, stage, stageVersion); err != nil {
			return ctrl.Result{}, err
		}
		if err = usage.DeleteActivationUsage(ctx, r.Client, activationUsage); err != nil {
			return ctrl.Result{}, err
		}
		if err := activation.PatchVectorActivationStatus(ctx, r.Client, req.NamespacedName, metav1.Condition{Type: landscape.ActivationSucceeded, Status: metav1.ConditionTrue, Reason: landscape.ActivationSucceeded, Message: fmt.Sprintf("successfully reconciled VectorActivation %s", vectorActivation.Name)}); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("VectorActivation reconciled successfully, set status to succeeded")
	}

	return ctrl.Result{}, nil
}

func (r *VectorActivationReconciler) checkExecutionsStatusAndPatchOnFailure(ctx context.Context, req ctrl.Request, executionsInActivation *landscape.ActivationTaskExecutionList, log logr.Logger) (bool, error) {
	allExecutionsSucceeded := true

	for _, exec := range executionsInActivation.Items {
		if meta.IsStatusConditionTrue(exec.Status.Conditions, landscape.ActivationTaskExecutionFailed) {
			msg := fmt.Sprintf("ActivationTaskExecution failed: %s", exec.Name)
			log.Info(msg)
			if err := activation.PatchVectorActivationStatus(ctx, r.Client, req.NamespacedName, metav1.Condition{Type: landscape.ActivationFailed, Status: metav1.ConditionTrue, Reason: landscape.ActivationTaskExecutionFailed, Message: msg}); err != nil {
				return false, err
			}
			allExecutionsSucceeded = false
		}
		if !meta.IsStatusConditionTrue(exec.Status.Conditions, landscape.ActivationTaskExecutionSucceeded) {
			allExecutionsSucceeded = false
			break
		}
	}
	return allExecutionsSucceeded, nil
}

func AcquireLease(ctx context.Context, c client.Client, controllerId string, namespace string, vectorActivation *landscape.VectorActivation, stage *common.Stage) (bool, error) {
	ownerRef := metav1.OwnerReference{
		APIVersion: vectorActivation.APIVersion,
		Kind:       vectorActivation.Kind,
		Name:       vectorActivation.Name,
		UID:        vectorActivation.UID,
	}
	return leaseLock.AcquireResourceLease(ctx, c, string(vectorActivation.UID), namespace, controllerId, landscape.VectorActivationKind, stage.Name, ownerRef)
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

	var stageOwnerRef *metav1.OwnerReference
	for _, owner := range stageVersion.OwnerReferences {
		if owner.Kind == common.StageKind && owner.APIVersion == common.GroupVersion.String() {
			stageOwnerRef = &owner
			break
		}
	}
	if stageOwnerRef == nil {
		return nil, nil, nil, fmt.Errorf("stage version %s does not have a stage owner reference", stageVersion.Name)
	}

	stage := &common.Stage{}
	if err := r.Get(ctx, types.NamespacedName{Name: stageOwnerRef.Name, Namespace: req.Namespace}, stage); err != nil {
		return nil, nil, nil, fmt.Errorf("could not get stage: %w", err)
	}

	return vectorActivation, stageVersion, stage, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorActivationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ControllerID = os.Getenv("POD_NAME")
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorActivation{}).
		Named("vectoractivation").
		Owns(&landscape.ActivationTaskExecution{},
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc:  func(e event.UpdateEvent) bool { return true },
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				CreateFunc:  func(e event.CreateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			})).
		Complete(r)
}

func (r *VectorActivationReconciler) cleanupVectorActivation(ctx context.Context, req ctrl.Request, vectorActivation *landscape.VectorActivation, stageName string) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("release lease for vector vectorActivation")
	if err := leaseLock.ReleaseResourceLease(ctx, r.Client, string(vectorActivation.UID), req.Namespace, r.ControllerID, landscape.VectorActivationKind, stageName); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to release lease: %w", err)
	}
	return ctrl.Result{}, nil
}
