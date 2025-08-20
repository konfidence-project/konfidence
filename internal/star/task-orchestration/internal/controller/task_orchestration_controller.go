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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// TaskOrchestrationReconciler reconciles a VectorMigration object
type TaskOrchestrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectormigrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectormigrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stageversionusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stageversionusages/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *TaskOrchestrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling vectorMigration")

	// get vectorMigration
	vectorMigration := &landscape.VectorMigration{}
	if err := r.Get(ctx, req.NamespacedName, vectorMigration); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "Unable to fetch vectorMigration")
			return ctrl.Result{}, err
		}
	}

	// get stageVersion
	stageVersion := &landscape.StageVersion{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      vectorMigration.Spec.StageVersion,
		Namespace: req.Namespace,
	}, stageVersion); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "Unable to fetch stageVersion")
			return ctrl.Result{}, err
		}
	}

	// check if a stageVersionUsage already exists
	stageVersionUsage := &landscape.StageVersionUsage{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      getStageVersionUsageName(stageVersion),
		Namespace: req.Namespace,
	}, stageVersionUsage); err != nil {
		if errors.IsNotFound(err) {
			// create new stageVersionUsage
			stageVersionUsage, err = constructStageVersionUsage(r, vectorMigration, stageVersion)
			if err != nil {
				log.Error(err, "Unable to construct vectorDeployment from template")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, stageVersionUsage); err != nil {
				log.Error(err, "Unable to create stageVersionUsage", "stageVersionUsage", stageVersionUsage)
				return ctrl.Result{}, err
			}

			log.V(1).Info("Created stageVersionUsage", "stageVersionUsage", stageVersionUsage)
			log.V(1).Info("Set stageVersionUsage owner for stageVersion")

			// set stageVersionUsage as owner of the stageVersion
			if err := controllerutil.SetOwnerReference(stageVersionUsage, stageVersion, r.Scheme); err != nil {
				log.Error(err, "Failed to add stageVersionUsage ownerRef to stageVersion")
				return ctrl.Result{}, err
			}

			if err := r.Update(ctx, stageVersion); err != nil {
				log.Error(err, "Failed to update stageVersion owner references")
				return ctrl.Result{}, err
			}

			log.Info("Successfully set stageVersionUsage as owner of stageVersion")
		} else {
			log.Error(err, "Unable to fetch stageVersionUsage")
			return ctrl.Result{}, err
		}
	}

	// TODO should we validate the stageVersionUsage for correct ownership here?

	log.Info("VectorMigration reconciled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskOrchestrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorMigration{}).
		Named("taskOrchestration").
		Complete(r)
}

func constructStageVersionUsage(r *TaskOrchestrationReconciler, vectorMigration *landscape.VectorMigration, stageVersion *landscape.StageVersion) (*landscape.StageVersionUsage, error) {
	stageVersionUsage := &landscape.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getStageVersionUsageName(stageVersion),
			Namespace: stageVersion.Namespace,
		},
	}

	// set vectorMigration as owner of the stageVersionUsage
	if err := controllerutil.SetOwnerReference(vectorMigration, stageVersionUsage, r.Scheme); err != nil {
		return nil, err
	}

	return stageVersionUsage, nil
}

func getStageVersionUsageName(stageVersion *landscape.StageVersion) string {
	return fmt.Sprintf("%s-%s", stageVersion.Name, "usage")
}
