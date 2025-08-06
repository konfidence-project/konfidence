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
	"strings"
	"time"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// StageReconciler reconciles a Stage object
type StageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	retryInterval = 30 * time.Second
)

// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeploymentusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeploymentusages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployment,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployment/status,verbs=get;update;patch

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

	adaptedVectorName, err := adaptVectorName(stage.Spec.Vector)
	if err != nil {
		return ctrl.Result{}, err
	}

	// get all existing vectorDeploymentUsages for this stage
	vectorDeploymentUsages := &landscape.VectorDeploymentUsageList{}
	if err := r.List(ctx, vectorDeploymentUsages, client.InNamespace(req.Namespace), client.MatchingFields{vectorDeploymentUsageOwnerKey: req.Name}); err != nil {
		log.Error(err, "Unable to list vectorDeploymentUsages")
		return ctrl.Result{}, err
	}

	// check if a vectorDeploymentUsage exists with a vector matching the stage vector
	index := slices.IndexFunc(vectorDeploymentUsages.Items, func(usage landscape.VectorDeploymentUsage) bool {
		return usage.Spec.Vector == stage.Spec.Vector
	})

	var vectorDeploymentUsage *landscape.VectorDeploymentUsage

	// create it if it does not exist
	if index < 0 {
		log.V(1).Info("No matching vectorDeploymentUsage found. Creating a new one...")

		// create new vectorDeploymentUsage
		vectorDeploymentUsage, err := constructVectorDeploymentUsageForStage(r, stage)
		if err != nil {
			log.Error(err, "Unable to construct vectorDeploymentUsage from template")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, vectorDeploymentUsage); err != nil {
			log.Error(err, "Unable to create vectorDeploymentUsage for stage", "vectorDeploymentUsage", vectorDeploymentUsage)
			return ctrl.Result{}, err
		}

		log.V(1).Info("Created vectorDeploymentUsage for stage", "vectorDeploymentUsage", vectorDeploymentUsage)
	} else {
		log.V(1).Info("Found existing vectorDeploymentUsage at index", "index", index)
		vectorDeploymentUsage = &vectorDeploymentUsages.Items[index]
	}

	oldVectorDeployments := map[string]bool{}

	// then delete all other obsolete vectorDeploymentUsages for this stage
	for i, usage := range vectorDeploymentUsages.Items {
		if i == index {
			// skip existing one if available
			continue
		}

		// only delete vectorDeploymentUsages that do not reference the current stage vector
		if usage.Spec.Vector != stage.Spec.Vector {
			// has the vectorDeployment already been processed?
			if !oldVectorDeployments[usage.Spec.Vector] {
				adaptedVectorName, err := adaptVectorName(usage.Spec.Vector)
				if err != nil {
					return ctrl.Result{}, err
				}

				// check if there is a vectorDeployment for an old stageVector
				vectorDeployment := &landscape.VectorDeployment{}
				if err := r.Get(ctx, types.NamespacedName{
					Namespace: stage.Namespace,
					Name:      adaptedVectorName,
				}, vectorDeployment); err != nil {
					if errors.IsNotFound(err) {
						// mark vectorDeployment as processed
						oldVectorDeployments[usage.Spec.Vector] = true
					} else {
						log.Error(err, "Unable to get old vectorDeployment for vectorDeploymentUsage", "vectorDeploymentUsage", usage)
						return ctrl.Result{}, err
					}
				}

				// check if the vectorDeployment still has an owner references to the stage
				exists, err := controllerutil.HasOwnerReference(vectorDeployment.GetOwnerReferences(), stage, r.Scheme)
				if err != nil {
					log.Error(err, "Unable to check owner references of vectorDeployment", "vectorDeployment", vectorDeployment)
					return ctrl.Result{}, err
				}

				// remove stage owner reference if it exists
				if exists {
					if err = controllerutil.RemoveOwnerReference(stage, vectorDeployment, r.Scheme); err != nil {
						log.Error(err, "Unable to remove stage owner reference of vectorDeployment", "vectorDeployment", vectorDeployment)
						return ctrl.Result{}, err
					}

					if err = r.Update(ctx, vectorDeployment); err != nil {
						log.Error(err, "Failed to update old vectorDeployment", "vectorDeployment", vectorDeployment)
						return ctrl.Result{}, err
					}

				}

				// mark vectorDeployment as processed
				oldVectorDeployments[usage.Spec.Vector] = true
			}

			if err := r.Delete(ctx, &usage, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "Unable to delete old vectorDeploymentUsage", "vectorDeploymentUsage", usage)
			} else {
				log.V(1).Info("Deleted old vectorDeploymentUsage", "vectorDeploymentUsage", usage)
			}
		}
	}

	// check if a vectorDeployment exists matching the stage vector
	vectorDeployment := &landscape.VectorDeployment{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: stage.Namespace,
		Name:      adaptedVectorName,
	}, vectorDeployment)

	if err != nil {
		if errors.IsNotFound(err) {
			// create a new one
			log.V(1).Info("No matching vectorDeployment found. Creating a new one...")

			// create new vectorDeployment
			vectorDeployment, err = constructVectorDeployment(r, stage, vectorDeploymentUsage)
			if err != nil {
				log.Error(err, "Unable to construct vectorDeployment from template")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, vectorDeployment); err != nil {
				log.Error(err, "Unable to create vectorDeployment", "vectorDeployment", vectorDeployment)
				return ctrl.Result{}, err
			}

			log.V(1).Info("Created vectorDeployment", "vectorDeployment", vectorDeployment)

			// update status to VectorDeploymentCreated
			meta.SetStatusCondition(&stage.Status.Conditions, metav1.Condition{Type: common.VectorDeploymentCreatedCondition,
				Status: metav1.ConditionTrue, Reason: common.VectorDeploymentCreatedCondition,
				Message: fmt.Sprintf("Successfully created VectorDeployment %s for stage %s", vectorDeployment.Name, stage.Name)})

			if err := r.Status().Update(ctx, stage); err != nil {
				log.Error(err, "Failed to update stage status")
				return ctrl.Result{}, err
			}

		} else {
			log.Error(err, "Unable to fetch vectorDeployment")
			return ctrl.Result{}, err
		}
	}

	log.V(1).Info("Found existing vectorDeployment")

	// get latest vectorDeployment
	err = r.Get(ctx, types.NamespacedName{
		Namespace: stage.Namespace,
		Name:      adaptedVectorName,
	}, vectorDeployment)

	if err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("Set stage owner for vectorDeployment")

	// set stage as owner
	if err := controllerutil.SetOwnerReference(stage, vectorDeployment, r.Scheme); err != nil {
		log.Error(err, "Failed to add stage ownerRef to vectorDeployment")
		return ctrl.Result{}, err
	}

	log.V(1).Info("Set vectorDeploymentUsage owner for vectorDeployment")

	// set vectorDeploymentUsage as owner
	if err := controllerutil.SetOwnerReference(vectorDeploymentUsage, vectorDeployment, r.Scheme); err != nil {
		log.Error(err, "Failed to add vectorDeploymentUsage ownerRef to vectorDeployment")
		return ctrl.Result{}, err
	}

	log.V(1).Info("Update owner references")
	if err := r.Update(ctx, vectorDeployment); err != nil {
		log.Error(err, "Failed to update vectorDeployment owner references")
		return ctrl.Result{}, err
	}

	// get latest vectorDeployment
	err = r.Get(ctx, types.NamespacedName{
		Namespace: stage.Namespace,
		Name:      adaptedVectorName,
	}, vectorDeployment)

	if err != nil {
		return ctrl.Result{}, err
	}

	// check if vectorDeployment is marked as deployed
	if meta.FindStatusCondition(vectorDeployment.Status.Conditions, landscape.VectorDeployedCondition) != nil {
		// if so the stage is marked as ready as well
		meta.SetStatusCondition(&stage.Status.Conditions, metav1.Condition{Type: common.StageReady,
			Status: metav1.ConditionTrue, Reason: common.StageReady,
			Message: fmt.Sprintf("Stage %s reconciled successfully", stage.Name)})

		if err := r.Status().Update(ctx, stage); err != nil {
			log.Error(err, "Failed to update stage status")
			return ctrl.Result{}, err
		}

		log.Info("Stage reconciled")
		return ctrl.Result{}, nil
	} else {
		// retry after some time
		return ctrl.Result{RequeueAfter: retryInterval}, nil
	}
}

var (
	vectorDeploymentUsageOwnerKey = ".metadata.controller"
	apiGVStr                      = common.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *StageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &landscape.VectorDeploymentUsage{}, vectorDeploymentUsageOwnerKey, func(rawObj client.Object) []string {
		// grab the vectorDeploymentUsage object and extract the owner
		vectorDeploymentUsage := rawObj.(*landscape.VectorDeploymentUsage)
		owner := metav1.GetControllerOf(vectorDeploymentUsage)
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
		Owns(&landscape.VectorDeploymentUsage{}).
		Watches(
			&landscape.VectorDeployment{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				// get all stage owners of the vector deployment
				refs := obj.GetOwnerReferences()
				var stageRefs []metav1.OwnerReference
				for i := range refs {
					if refs[i].Kind == common.StageKind {
						stageRefs = append(stageRefs, refs[i])
					}
				}

				// call reconciliation for each stage owner
				requests := make([]reconcile.Request, 0, len(stageRefs))
				for i := range stageRefs {
					requests = append(requests,
						reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      stageRefs[i].Name,
								Namespace: obj.GetNamespace(),
							},
						})
				}

				return requests
			}),
		).
		Named("stage").
		Complete(r)
}

func constructVectorDeploymentUsageForStage(r *StageReconciler, stage *common.Stage) (*landscape.VectorDeploymentUsage, error) {
	name := fmt.Sprintf("%s-%s-%s", "vectordeploymentusage", stage.Name, rand.String(6))
	vectorDeploymentUsage := &landscape.VectorDeploymentUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: stage.Namespace,
		},
		Spec: landscape.VectorDeploymentUsageSpec{
			Vector: stage.Spec.Vector,
		},
	}

	// set stage as owner and controller
	if err := ctrl.SetControllerReference(stage, vectorDeploymentUsage, r.Scheme); err != nil {
		return nil, err
	}

	return vectorDeploymentUsage, nil
}

func constructVectorDeployment(r *StageReconciler, stage *common.Stage, vectorDeploymentUsage *landscape.VectorDeploymentUsage) (*landscape.VectorDeployment, error) {
	adaptedVectorName, err := adaptVectorName(stage.Spec.Vector)
	if err != nil {
		return nil, err
	}

	vectorDeployment := &landscape.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      adaptedVectorName,
			Namespace: stage.Namespace,
		},
		Spec: landscape.VectorDeploymentSpec{
			Vector: stage.Spec.Vector,
		},
	}

	// set usage as owner
	if err := controllerutil.SetOwnerReference(vectorDeploymentUsage, vectorDeployment, r.Scheme); err != nil {
		return nil, err
	}

	// set stage as owner
	if err := controllerutil.SetOwnerReference(stage, vectorDeployment, r.Scheme); err != nil {
		return nil, err
	}

	return vectorDeployment, nil
}

// make vector name usable as kubernetes resource name
func adaptVectorName(vector string) (string, error) {
	trimmedVector := strings.TrimSpace(strings.ToLower(vector))

	// TODO validate defined vector format
	if len(trimmedVector) < 4 {
		return "", fmt.Errorf("unable to parse vector: %s", vector)
	}

	// get index of separator
	separatorIdx := strings.LastIndex(trimmedVector, "//")

	if separatorIdx == -1 || separatorIdx == len(vector)-2 {
		return "", fmt.Errorf("unable to parse vector: %s", vector)
	}

	componentVersion := trimmedVector[separatorIdx+2:]
	adaptedVector := strings.ReplaceAll(componentVersion, "/", ".")
	adaptedVector = strings.ReplaceAll(adaptedVector, ":", "-")
	return adaptedVector, nil
}
