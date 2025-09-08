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
	"strings"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// StageVersionReconciler reconciles a StageVersion object
type StageVersionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stageversions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectormigrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectormigrations/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *StageVersionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stageVersion")

	// get stageVersion
	stageVersion := &landscape.StageVersion{}
	if err := r.Get(ctx, req.NamespacedName, stageVersion); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "Unable to fetch stageVersion")
			return ctrl.Result{}, err
		}
	}

	adaptedVectorName, err := adaptVectorName(stageVersion.Spec.Vector)
	if err != nil {
		return ctrl.Result{}, err
	}

	// check if a vectorDeployment exists matching the stage vector
	vectorDeployment := &landscape.VectorDeployment{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: stageVersion.Namespace,
		Name:      adaptedVectorName,
	}, vectorDeployment)

	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("No matching vectorDeployment found. Creating a new one...")

			// create new vectorDeployment
			vectorDeployment, err = constructVectorDeployment(r, stageVersion)
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
			meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{Type: landscape.VectorDeploymentCreatedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDeploymentCreatedCondition,
				Message: fmt.Sprintf("Successfully created VectorDeployment %s for stageVersion %s", vectorDeployment.Name, stageVersion.Name)})

			if err := r.Status().Update(ctx, stageVersion); err != nil {
				log.Error(err, "Failed to update stageVersion status")
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
		Namespace: stageVersion.Namespace,
		Name:      adaptedVectorName,
	}, vectorDeployment)

	if err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("Set stageVersion owner for vectorDeployment")

	// set stageVersion as owner
	if err := controllerutil.SetOwnerReference(stageVersion, vectorDeployment, r.Scheme); err != nil {
		log.Error(err, "Failed to add stageVersion ownerRef to vectorDeployment")
		return ctrl.Result{}, err
	}

	log.V(1).Info("Update owner references")
	if err := r.Update(ctx, vectorDeployment); err != nil {
		log.Error(err, "Failed to update vectorDeployment owner references")
		return ctrl.Result{}, err
	}

	// get latest vectorDeployment
	err = r.Get(ctx, types.NamespacedName{
		Namespace: stageVersion.Namespace,
		Name:      adaptedVectorName,
	}, vectorDeployment)

	if err != nil {
		return ctrl.Result{}, err
	}

	// check if vectorDeployment is marked as deployed
	if !meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, landscape.VectorDeployedCondition) {
		// wait for vectorDeployment status change notification
		return ctrl.Result{}, nil
	}

	// check if a vectorMigration for this stageVersion already exists
	vectorMigration := &landscape.VectorMigration{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: stageVersion.Namespace,
		Name:      getVectorMigrationName(stageVersion),
	}, vectorMigration)

	if err != nil {
		if errors.IsNotFound(err) {
			log.V(1).Info("No matching vectorMigration found. Creating a new one...")

			// create new vectorMigration
			vectorMigration, err = constructVectorMigration(r, stageVersion)
			if err != nil {
				log.Error(err, "Unable to construct vectorMigration from template")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, vectorMigration); err != nil {
				log.Error(err, "Unable to create vectorMigration", "vectorMigration", vectorMigration)
				return ctrl.Result{}, err
			}

			log.V(1).Info("Created vectorMigration", "vectorMigration", vectorMigration)

			// update status to VectorMigrationCreated
			meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{Type: landscape.VectorMigrationCreatedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorMigrationCreatedCondition,
				Message: fmt.Sprintf("Successfully created vectorMigration %s for stageVersion %s", vectorMigration.Name, stageVersion.Name)})

			if err := r.Status().Update(ctx, stageVersion); err != nil {
				log.Error(err, "Failed to update stageVersion status")
				return ctrl.Result{}, err
			}
		} else {
			log.Error(err, "Unable to fetch vectorMigration")
			return ctrl.Result{}, err
		}
	}

	log.V(1).Info("Found existing vectorMigration")

	// get stageVersion
	if err := r.Get(ctx, req.NamespacedName, stageVersion); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "Unable to fetch stageVersion")
			return ctrl.Result{}, err
		}
	}

	// mark the stage version as ready
	meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{Type: landscape.StageVersionReady,
		Status: metav1.ConditionTrue, Reason: landscape.StageVersionReady,
		Message: fmt.Sprintf("StageVersion %s reconciled successfully", stageVersion.Name)})

	if err := r.Status().Update(ctx, stageVersion); err != nil {
		log.Error(err, "Failed to update stageVersion status")
		return ctrl.Result{}, err
	}

	log.Info("StageVersion reconciled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.StageVersion{}).
		Owns(&landscape.VectorMigration{}).
		Watches(
			&landscape.VectorDeployment{},
			handler.EnqueueRequestsFromMapFunc(reconcileStageVersionOwner),
		).
		Named("stageVersion").
		Complete(r)
}

func constructVectorDeployment(r *StageVersionReconciler, stageVersion *landscape.StageVersion) (*landscape.VectorDeployment, error) {
	adaptedVectorName, err := adaptVectorName(stageVersion.Spec.Vector)
	if err != nil {
		return nil, err
	}

	vectorDeployment := &landscape.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      adaptedVectorName,
			Namespace: stageVersion.Namespace,
		},
		Spec: landscape.VectorDeploymentSpec{
			Vector: stageVersion.Spec.Vector,
		},
	}

	// set stageVersion as owner
	if err := controllerutil.SetOwnerReference(stageVersion, vectorDeployment, r.Scheme); err != nil {
		return nil, err
	}
	return vectorDeployment, nil
}

func constructVectorMigration(r *StageVersionReconciler, stageVersion *landscape.StageVersion) (*landscape.VectorMigration, error) {
	vectorMigration := &landscape.VectorMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getVectorMigrationName(stageVersion),
			Namespace: stageVersion.Namespace,
		},
		Spec: landscape.VectorMigrationSpec{
			Vector:       stageVersion.Spec.Vector,
			StageVersion: stageVersion.Name,
		},
	}

	// set stageVersion as controller
	if err := controllerutil.SetControllerReference(stageVersion, vectorMigration, r.Scheme); err != nil {
		return nil, err
	}
	return vectorMigration, nil
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

func reconcileStageVersionOwner(ctx context.Context, obj client.Object) []reconcile.Request {
	// get all stageVersion owners of the watched object
	refs := obj.GetOwnerReferences()
	var stageVersionRefs []metav1.OwnerReference
	for i := range refs {
		if refs[i].Kind == landscape.StageVersionKind {
			stageVersionRefs = append(stageVersionRefs, refs[i])
		}
	}

	// call reconciliation for each stageVersion owner
	requests := make([]reconcile.Request, 0, len(stageVersionRefs))
	for i := range stageVersionRefs {
		requests = append(requests,
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      stageVersionRefs[i].Name,
					Namespace: obj.GetNamespace(),
				},
			})
	}

	return requests
}

func getVectorMigrationName(stageVersion *landscape.StageVersion) string {
	return fmt.Sprintf("%s-%s", stageVersion.Name, "migration")
}
