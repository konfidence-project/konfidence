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

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// VectorActivationReconciler reconciles a VectorActivation object
type VectorActivationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations/finalizers,verbs=update

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
	log.Info("Reconciling vectorActivation", "namespace", req.Namespace, "name", req.Name)

	vectorActivation := &landscape.VectorActivation{}
	if err := r.Get(ctx, req.NamespacedName, vectorActivation); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	stageVersion := &landscape.StageVersion{}
	if err := r.Get(ctx, types.NamespacedName{Name: vectorActivation.Spec.StageVersion, Namespace: req.Namespace}, stageVersion); err != nil {
		return ctrl.Result{}, fmt.Errorf("could not get stage version: %w", err)
	}

	if meta.IsStatusConditionTrue(vectorActivation.Status.Conditions, landscape.VectorActivationSucceeded) {
		log.Info("VectorActivation already succeeded, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// TODO: acquire lease

	// TODO: requeue and wait if lease is held by another process

	//TODO: add payload to execution CR
	activationExecution := &landscape.ActivationExecution{ObjectMeta: metav1.ObjectMeta{Name: vectorActivation.Name + "-execution", Namespace: req.Namespace}, Spec: landscape.ActivationExecutionSpec{Type: "gateway-api-http-route", Name: "example"}}
	if err := r.Create(ctx, activationExecution); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create ActivationExecution: %w ", err)
	}

	log.Info("Finished processing ActivationExecution")

	// TODO: monitor activationExecution CR

	// TODO: If all activationExecutions are complete:
	//     update status to Ready
	//     release lease

	// update status if activationExecution succeeded
	if meta.IsStatusConditionTrue(activationExecution.Status.Conditions, landscape.ActivationExecutionSucceeded) {
		if err := r.updateVectorActivationStatus(ctx, req, metav1.Condition{Type: landscape.VectorActivationSucceeded, Status: metav1.ConditionTrue, Reason: landscape.VectorActivationSucceeded, Message: fmt.Sprintf("successfully reconciled VectorActivation %s", vectorActivation.Name)}); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update VectorActivation status: %w ", err)
		}
		if err := r.Status().Update(ctx, vectorActivation); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update VectorActivation status: %w ", err)
		}
		log.Info("VectorActivation succeeded")
	}

	log.Info("VectorActivation reconciled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorActivationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorActivation{}).
		Owns(&landscape.ActivationExecution{}).
		Named("vectoractivation").
		Complete(r)
}

func (r *VectorActivationReconciler) updateVectorActivationStatus(ctx context.Context, req ctrl.Request, condition metav1.Condition) error {
	vectorActivation := &landscape.VectorActivation{}
	if err := r.Get(ctx, req.NamespacedName, vectorActivation); err != nil {
		return fmt.Errorf("unable to fetch vectorActivation: %w", err)
	}

	meta.SetStatusCondition(&vectorActivation.Status.Conditions, condition)

	if err := r.Status().Update(ctx, vectorActivation); err != nil {
		return fmt.Errorf("unable to update vectorActivation status: %w", err)
	}

	return nil
}
