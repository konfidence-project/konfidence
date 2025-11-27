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

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	leaseLock "github.com/konfidence-project/landscape-vector-activation-controller/internal/lock"
	"github.com/konfidence-project/landscape-vector-activation-controller/internal/usages"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// VectorActivationReconciler reconciles a VectorActivation object
type VectorActivationReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Config       *rest.Config
	ClientSet    *kubernetes.Clientset
	ControllerID string
}

// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=activationexecutions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=activationexecutions/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VectorActivation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *VectorActivationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("VectorActivation reconcile started...")

	vectorActivation := &landscape.VectorActivation{}
	if err := r.Get(ctx, req.NamespacedName, vectorActivation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	stageVersion := &landscape.StageVersion{}
	if err := r.Get(ctx, types.NamespacedName{Name: vectorActivation.Spec.StageVersion, Namespace: req.Namespace}, stageVersion); err != nil {
		return ctrl.Result{}, fmt.Errorf("could not get stage version: %w", err)
	}

	var stageOwnerRef *metav1.OwnerReference
	for _, owner := range stageVersion.OwnerReferences {
		if owner.Kind == common.StageKind && owner.APIVersion == common.GroupVersion.String() {
			stageOwnerRef = &owner
			break
		}
	}
	if stageOwnerRef == nil {
		return ctrl.Result{}, fmt.Errorf("stage version %s does not have a stage owner reference", stageVersion.Name)
	}

	stage := &common.Stage{}
	if err := r.Get(ctx, types.NamespacedName{Name: stageOwnerRef.Name, Namespace: req.Namespace}, stage); err != nil {
		return ctrl.Result{}, fmt.Errorf("could not get stage: %w", err)
	}

	if meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, landscape.ActivationSucceeded) {
		return r.cleanupVectorActivation(ctx, req, vectorActivation, stage.Name)
	}

	acquired, err := r.acquireLease(ctx, req, vectorActivation, stage)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to acquire lease: %w", err)
	}
	if !acquired {
		log.Info("Lease not acquired, requeuing")
		return ctrl.Result{RequeueAfter: leaseLock.DefaultLeaseTTL}, nil
	}
	log.Info("Lease acquired", "acquired", acquired)

	activeStageVersionUsage, err := usages.GetCurrentActiveUsage(ctx, r.Client, stage)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get current active usage: %w", err)
	}

	if activeStageVersionUsage != nil {
		isNewer, err := usages.IsNewerThanCurrentActiveUsage(ctx, r.Client, stageVersion, activeStageVersionUsage)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to compare stage versions: %w", err)
		}
		if !isNewer {
			log.Info("activation belongs to older stage version than currently active one, skipping")
			// TODO: update status to skipped and release lease

			return ctrl.Result{}, nil
		}
	}

	activationUsage, err := usages.CreateOrUpdateActivationUsage(ctx, r.Client, stage, stageVersion, vectorActivation)
	if err != nil {
		return ctrl.Result{}, err
	}

	// TODO: handle executions here

	if err := usages.CreateOrUpdateActiveUsage(ctx, r.Client, activeStageVersionUsage, stage, stageVersion); err != nil {
		return ctrl.Result{}, err
	}

	if err = usages.DeleteActivationUsage(ctx, r.Client, activationUsage); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.patchVectorActivationStatus(ctx, req, metav1.Condition{Type: landscape.ActivationSucceeded, Status: metav1.ConditionTrue, Reason: landscape.ActivationSucceeded, Message: fmt.Sprintf("successfully reconciled VectorActivation %s", vectorActivation.Name)}); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update VectorActivation status: %w", err)
	}
	log.Info("VectorActivation reconciled successfully, set status to succeeded")

	return ctrl.Result{}, nil
}

func (r *VectorActivationReconciler) acquireLease(ctx context.Context, req ctrl.Request, vectorActivation *landscape.VectorActivation, stage *common.Stage) (bool, error) {
	ownerRef := metav1.OwnerReference{
		APIVersion: vectorActivation.APIVersion,
		Kind:       vectorActivation.Kind,
		Name:       vectorActivation.Name,
		UID:        vectorActivation.UID,
	}
	return leaseLock.AcquireResourceLease(ctx, r.ClientSet, string(vectorActivation.UID), req.Namespace, r.ControllerID, landscape.VectorActivationKind, stage.Name, ownerRef)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorActivationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ControllerID = os.Getenv("POD_NAME")
	r.ClientSet = kubernetes.NewForConfigOrDie(r.Config)
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorActivation{}).
		Named("vectoractivation").
		Complete(r)
}

func (r *VectorActivationReconciler) patchVectorActivationStatus(ctx context.Context, req ctrl.Request, condition metav1.Condition) error {
	vectorActivation := &landscape.VectorActivation{}
	if err := r.Get(ctx, req.NamespacedName, vectorActivation); err != nil {
		return fmt.Errorf("unable to fetch vectorActivation: %w", err)
	}
	oldState := vectorActivation.DeepCopy()
	meta.SetStatusCondition(&vectorActivation.Status.Conditions, condition)
	return r.Status().Patch(ctx, vectorActivation, client.MergeFrom(oldState))
}

func (r *VectorActivationReconciler) cleanupVectorActivation(ctx context.Context, req ctrl.Request, activation *landscape.VectorActivation, stageName string) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("cleanup vector activation")

	log.Info("release lease for vector activation")
	if err := leaseLock.ReleaseResourceLease(ctx, r.ClientSet, string(activation.UID), req.Namespace, r.ControllerID, landscape.VectorActivationKind, stageName); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to release lease: %w", err)
	}

	log.Info("vector activation cleaned up")
	return ctrl.Result{}, nil
}
