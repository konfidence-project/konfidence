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
	"reflect"
	"slices"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	util "github.com/konfidence-project/landscape-stage-controller/internal/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
	log.Info("Reconcile stage started...")

	// get stage
	stage := &common.Stage{}
	if err := r.Get(ctx, req.NamespacedName, stage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, r.reconcileStage(ctx, req, stage)
}

func (r *StageReconciler) reconcileStage(ctx context.Context, req ctrl.Request, stage *common.Stage) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stage")

	_, err := r.getOrCreateTargetStageVersionUsage(ctx, req, stage)
	if err != nil {
		return err
	}

	_, err = r.getOrCreateStageVersion(ctx, req, stage)
	if err != nil {
		return err
	}

	log.Info("Stage reconciled")
	return nil
}

func (r *StageReconciler) getOrCreateTargetStageVersionUsage(ctx context.Context, req ctrl.Request, stage *common.Stage) (*landscape.StageVersionUsage, error) {
	log := logf.FromContext(ctx)

	stageVersionUsages := &landscape.StageVersionUsageList{}
	if err := r.List(ctx, stageVersionUsages, client.InNamespace(req.Namespace), client.MatchingLabels(getTargetStageVersionUsageLabels(stage))); err != nil {
		return nil, fmt.Errorf("unable to list current target stageVersionUsages: %w", err)
	}

	if len(stageVersionUsages.Items) == 0 {
		log.Info("No matching target stageVersionUsage found. Creating a new one...")

		// create new stageVersionUsage
		stageVersionUsage, err := r.constructStageVersionUsage(stage)
		if err != nil {
			return nil, fmt.Errorf("unable to construct target stageVersionUsage from template: %w", err)
		}

		if err := r.Create(ctx, stageVersionUsage); err != nil {
			return nil, fmt.Errorf("unable to create target stageVersionUsage: %w", err)
		}

		log.Info("Created target stageVersionUsage", "stageVersionUsage", stageVersionUsage)
		return stageVersionUsage, nil
	}

	// found one or more matching target stageVersionUsages
	// we just use the first one found
	stageVersionUsage := &stageVersionUsages.Items[0]
	log.Info("Found existing target stageVersionUsage", "stageVersionUsage", stageVersionUsage)

	// update the target usage with the current spec
	if err := r.updateTargetStageVersionUsage(ctx, stageVersionUsage, stage); err != nil {
		return nil, fmt.Errorf("unable to update target stageVersionUsage specs: %w", err)
	}

	if len(stageVersionUsages.Items) > 1 {
		log.Info("Deleting obsolete target stageVersionUsages")

		// delete all other (potentially manually) created target stageVersionUsages
		for _, stageVersionUsage := range stageVersionUsages.Items[1:] {
			if err := r.Delete(ctx, &stageVersionUsage); err != nil {
				return nil, fmt.Errorf("unable to delete obsolete target stageVersionUsage: %w", err)
			}
		}
	}

	return stageVersionUsage, nil
}

func (r *StageReconciler) getOrCreateStageVersion(ctx context.Context, req ctrl.Request, stage *common.Stage) (*landscape.StageVersion, error) {
	log := logf.FromContext(ctx)

	// get all stageVersions that are owned by this stage
	stageVersions := &landscape.StageVersionList{}
	if err := r.List(ctx, stageVersions, client.InNamespace(req.Namespace), client.MatchingFields{stageVersionOwnerKey: req.Name}); err != nil {
		return nil, fmt.Errorf("unable to list stageVersions: %w", err)
	}

	// check if a stageVersion exists with a vector matching the stage vector and the current stage generation
	index := slices.IndexFunc(stageVersions.Items, func(version landscape.StageVersion) bool {
		return version.Spec.Vector == stage.Spec.Vector && version.Spec.StageGeneration == stage.Generation
	})

	// create it if it does not exist
	if index < 0 {
		log.Info("No matching stageVersion found. Creating a new one...")

		// create new stageVersion
		stageVersion, err := r.constructStageVersion(stage)
		if err != nil {
			return nil, fmt.Errorf("unable to construct stageVersion from template: %w", err)
		}

		if err := r.Create(ctx, stageVersion); err != nil {
			return nil, fmt.Errorf("unable to create stageVersion for stage: %w", err)
		}

		log.Info("Created stageVersion for stage", "stageVersion", stageVersion)
		return stageVersion, nil
	} else {
		log.V(1).Info("Found existing stageVersion at index", "index", index)
		return &stageVersions.Items[index], nil
	}
}

func (r *StageReconciler) constructStageVersion(stage *common.Stage) (*landscape.StageVersion, error) {
	name := fmt.Sprintf("%s-%s-%s", "stage-version", stage.Name, rand.String(8))
	stageVersionLabels, err := getStageVersionLabels(stage)
	if err != nil {
		return nil, err
	}

	stageVersion := &landscape.StageVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stage.Namespace,
			Labels:    stageVersionLabels,
		},
		Spec: landscape.StageVersionSpec{
			Vector:          stage.Spec.Vector,
			StageGeneration: stage.Generation,
		},
	}

	if err := ctrl.SetControllerReference(stage, stageVersion, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for stageVersion: %w", err)
	}

	return stageVersion, nil
}

func (r *StageReconciler) constructStageVersionUsage(stage *common.Stage) (*landscape.StageVersionUsage, error) {
	name := fmt.Sprintf("%s-target-usage-%s", stage.Name, rand.String(8))
	stageVersionLabels, err := getStageVersionLabels(stage)
	if err != nil {
		return nil, err
	}

	stageVersionUsage := &landscape.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stage.Namespace,
			Labels:    getTargetStageVersionUsageLabels(stage),
		},
		Spec: landscape.StageVersionUsageSpec{
			Reason: StageVersionUsageTargetType,
			StageVersionSelector: &metav1.LabelSelector{
				MatchLabels: stageVersionLabels,
			},
		},
	}

	if err := controllerutil.SetControllerReference(stage, stageVersionUsage, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for stageVersionUsage: %w", err)
	}

	return stageVersionUsage, nil
}

func (r *StageReconciler) updateTargetStageVersionUsage(ctx context.Context, stageVersionUsage *landscape.StageVersionUsage, stage *common.Stage) error {
	stageVersionLabels, err := getStageVersionLabels(stage)
	if err != nil {
		return err
	}

	originalStageVersionUsage := stageVersionUsage.DeepCopy()
	patch := client.MergeFrom(originalStageVersionUsage)

	stageVersionUsage.Labels = getTargetStageVersionUsageLabels(stage)
	stageVersionUsage.Spec.Reason = StageVersionUsageTargetType
	stageVersionUsage.Spec.StageVersionRef = nil
	stageVersionUsage.Spec.StageVersionSelector = &metav1.LabelSelector{
		MatchLabels: stageVersionLabels,
	}

	// remove all owner references
	stageVersionUsage.OwnerReferences = []metav1.OwnerReference{}

	// and set stage controller reference
	if err := controllerutil.SetControllerReference(stage, stageVersionUsage, r.Scheme); err != nil {
		return fmt.Errorf("unable to set controller reference for stageVersionUsage: %w", err)
	}

	if !reflect.DeepEqual(stageVersionUsage, originalStageVersionUsage) {
		if err := r.Patch(ctx, stageVersionUsage, patch); err != nil {
			return fmt.Errorf("unable to patch target stageVersionUsage: %w", err)
		}
	}

	return nil
}

func getStageVersionLabels(stage *common.Stage) (map[string]string, error) {
	// TODO label and key value must have a max length of 63, cut vector name to last 63?
	adaptedVectorName, err := util.AdaptVectorName(stage.Spec.Vector)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		StageNameLabel:       stage.Name,
		VectorReferenceLabel: adaptedVectorName,
	}, nil
}

func getTargetStageVersionUsageLabels(stage *common.Stage) map[string]string {
	return map[string]string{
		StageVersionUsageTarget: stage.Name,
	}
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
		return fmt.Errorf("unable to create index for owner reference of stage version: %w", err)
	}

	noUpdatePredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},

		// Allow create events
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},

		// Allow delete events
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},

		// Allow generic events (e.g., external triggers)
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&common.Stage{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&landscape.StageVersion{}, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, noUpdatePredicate))).
		Owns(&landscape.StageVersionUsage{}, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, noUpdatePredicate))).
		Named("stage").
		Complete(r)
}
