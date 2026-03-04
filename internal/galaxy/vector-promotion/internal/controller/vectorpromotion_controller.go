/*
Copyright 2026.

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
	"reflect"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// VectorPromotionReconciler reconciles a VectorPromotion object
type VectorPromotionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectorpromotions/status,verbs=get;update;patch

func (r *VectorPromotionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile vectorPromotion started...")

	vectorPromotion := &global.VectorPromotion{}
	if err := r.Get(ctx, req.NamespacedName, vectorPromotion); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalVectorPromotion := vectorPromotion.DeepCopy()
	patch := client.MergeFrom(originalVectorPromotion)
	err := r.reconcileVectorPromotion(ctx, req, vectorPromotion)

	if !reflect.DeepEqual(vectorPromotion.Status, originalVectorPromotion.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorPromotion, patch); patchError != nil {
			patchErrorMessage := "unable to update vectorPromotion status"

			if err != nil {
				reconcileError := fmt.Errorf("an error occurred while reconciling vectorPromotion: %w", err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}

	return ctrl.Result{}, err
}

func (r *VectorPromotionReconciler) reconcileVectorPromotion(ctx context.Context, _ ctrl.Request, _ *global.VectorPromotion) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling vectorPromotion")

	// TODO implement

	log.Info("VectorPromotion reconciled")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorPromotionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&global.VectorPromotion{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("vectorPromotion").
		Complete(r)
}
