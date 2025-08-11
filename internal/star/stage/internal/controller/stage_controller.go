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
	"slices"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// StageReconciler reconciles a Stage object
type StageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversions/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *StageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stage")

	// get stage
	stage := &common.Stage{}
	if err := r.Get(ctx, req.NamespacedName, stage); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "Unable to fetch stage")
			return ctrl.Result{}, err
		}
	}

	// get all stageVersions that are owned by this stage
	stageVersions := &landscape.StageVersionList{}
	if err := r.List(ctx, stageVersions, client.InNamespace(req.Namespace), client.MatchingFields{stageVersionOwnerKey: req.Name}); err != nil {
		log.Error(err, "Unable to list stageVersions")
		return ctrl.Result{}, err
	}

	// check if a stageVersion exists with a vector matching the stage vector and the current stage generation
	index := slices.IndexFunc(stageVersions.Items, func(version landscape.StageVersion) bool {
		return version.Spec.Vector == stage.Spec.Vector && version.Spec.StageGeneration == stage.Generation
	})

	// create it if it does not exist
	if index < 0 {
		log.V(1).Info("No matching stageVersion found. Creating a new one...")

		// create new stageVersion
		stageVersion, err := constructStageVersionForStage(r, stage)
		if err != nil {
			log.Error(err, "Unable to construct stageVersion from template")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, stageVersion); err != nil {
			log.Error(err, "Unable to create stageVersion for stage", "stageVersion", stageVersion)
			return ctrl.Result{}, err
		}

		log.V(1).Info("Created stageVersion for stage", "stageVersion", stageVersion)
	} else {
		log.V(1).Info("Found existing stageVersion at index", "index", index)
	}

	for i, stageVersion := range stageVersions.Items {
		if i == index {
			// skip existing one if available
			continue
		}

		// check if the stageVersion still has an owner references to the stage
		exists, err := controllerutil.HasOwnerReference(stageVersion.GetOwnerReferences(), stage, r.Scheme)
		if err != nil {
			log.Error(err, "Unable to check owner references of stageVersion", "stageVersion", stageVersion)
			return ctrl.Result{}, err
		}

		if exists {
			// delete stageVersion if the stage owner ref is the last one remaining
			if len(stageVersion.GetOwnerReferences()) == 1 {
				log.V(1).Info("Removing old stageVersion because this stage owner reference is the last one remaining")

				if err := r.Delete(ctx, &stageVersion); err != nil {
					log.Error(err, "Unable to delete old stageVersion for stage", "stageVersion", stageVersion)
					return ctrl.Result{}, err
				}
			} else {
				// remove stage owner ref for this stage
				if err = controllerutil.RemoveOwnerReference(stage, &stageVersion, r.Scheme); err != nil {
					log.Error(err, "Unable to remove stage owner reference of stageVersion", "stageVersion", stageVersion)
					return ctrl.Result{}, err
				}

				if err = r.Update(ctx, &stageVersion); err != nil {
					log.Error(err, "Failed to update stageVersion", "stageVersion", stageVersion)
					return ctrl.Result{}, err
				}
			}
		}
	}

	log.Info("Stage reconciled")
	return ctrl.Result{}, nil
}

var (
	stageVersionOwnerKey = ".metadata.controller"
	apiGVStr             = common.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *StageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &landscape.StageVersion{}, stageVersionOwnerKey, func(rawObj client.Object) []string {
		// grab the stageVersion object and extract the owner
		stageVersion := rawObj.(*landscape.StageVersion)
		owner := metav1.GetControllerOf(stageVersion)
		if owner == nil {
			return nil
		}
		// make sure it is a stage...
		if owner.APIVersion != apiGVStr || owner.Kind != "Stage" {
			return nil
		}

		// and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&common.Stage{}).
		Owns(&landscape.StageVersion{}).
		Named("stage").
		Complete(r)
}

func constructStageVersionForStage(r *StageReconciler, stage *common.Stage) (*landscape.StageVersion, error) {
	name := fmt.Sprintf("%s-%s-%s", "stage-version", stage.Name, rand.String(8))
	stageVersion := &landscape.StageVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stage.Namespace,
		},
		Spec: landscape.StageVersionSpec{
			Vector:          stage.Spec.Vector,
			StageGeneration: stage.Generation,
		},
	}

	if err := ctrl.SetControllerReference(stage, stageVersion, r.Scheme); err != nil {
		return nil, err
	}

	return stageVersion, nil
}
