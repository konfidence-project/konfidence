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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectoractivations/status,verbs=get;update;patch

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
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// get a k8s conform vector name
	adaptedVectorName, err := adaptVectorName(stageVersion.Spec.Vector)
	if err != nil {
		return ctrl.Result{}, err
	}

	// check if a vectorDeployment exists matching the stage vector
	vectorDeployment, err := r.getOrCreateVectorDeployment(ctx, stageVersion, adaptedVectorName)
	if err != nil {
		return ctrl.Result{}, err
	}

	// set vectorDeploymentCreated status
	if err = r.setStageVersionStatusTrue(ctx, req, landscape.VectorDeploymentCreatedCondition, fmt.Sprintf("Successfully created VectorDeployment %s for stageVersion %s", vectorDeployment.Name, stageVersion.Name)); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.V(1).Info("Found existing vectorDeployment")

	// get latest vectorDeployment
	if err = r.Get(ctx, types.NamespacedName{Namespace: stageVersion.Namespace, Name: adaptedVectorName}, vectorDeployment); err != nil {
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
	if err = r.Get(ctx, types.NamespacedName{Namespace: stageVersion.Namespace, Name: adaptedVectorName}, vectorDeployment); err != nil {
		return ctrl.Result{}, err
	}

	// check if vectorDeployment is marked as deployed
	if !meta.IsStatusConditionTrue(vectorDeployment.Status.Conditions, landscape.VectorDeployedCondition) {
		// wait for vectorDeployment status change notification
		return ctrl.Result{}, nil
	}

	// check if a vectorMigration exists matching the stage vector
	vectorMigration, err := r.getOrCreateVectorMigration(ctx, stageVersion)
	if err != nil {
		return ctrl.Result{}, err
	}

	// set vectorMigrationCreated status
	if err = r.setStageVersionStatusTrue(ctx, req, landscape.VectorMigrationCreatedCondition, fmt.Sprintf("Successfully created vectorMigration %s for stageVersion %s", vectorMigration.Name, stageVersion.Name)); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.V(1).Info("Found existing vectorMigration")

	// check if vectorMigration is marked as successful
	if !meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, landscape.VectorMigrationSucceeded) {
		// wait for vectorMigration status change notification
		return ctrl.Result{}, nil
	}

	// set vectorMigrated status
	if err = r.setStageVersionStatusTrue(ctx, req, landscape.VectorMigratedCondition, fmt.Sprintf("VectorMigration %s successful for stageVersion %s", vectorMigration.Name, stageVersion.Name)); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// check if a vectorActivation exists matching the stage vector
	vectorActivation, err := r.getOrCreateVectorActivation(ctx, stageVersion)
	if err != nil {
		return ctrl.Result{}, err
	}

	// set vectorActivationCreated status
	if err = r.setStageVersionStatusTrue(ctx, req, landscape.VectorActivationCreatedCondition, fmt.Sprintf("Successfully created vectorActivation %s for stageVersion %s", vectorActivation.Name, stageVersion.Name)); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// set stageVersionReady status
	if err = r.setStageVersionStatusTrue(ctx, req, landscape.StageVersionReady, fmt.Sprintf("StageVersion %s reconciled successfully", stageVersion.Name)); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("StageVersion reconciled")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageVersionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.StageVersion{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&landscape.VectorMigration{}).
		Owns(&landscape.VectorActivation{}).
		Watches(
			&landscape.VectorDeployment{},
			handler.EnqueueRequestsFromMapFunc(reconcileStageVersionOwner),
		).
		Named("stageVersion").
		Complete(r)
}

func (r *StageVersionReconciler) getOrCreateVectorDeployment(ctx context.Context, stageVersion *landscape.StageVersion, adaptedVectorName string) (*landscape.VectorDeployment, error) {
	log := logf.FromContext(ctx)
	vectorDeployment := &landscape.VectorDeployment{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: stageVersion.Namespace,
		Name:      adaptedVectorName,
	}, vectorDeployment)

	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Unable to fetch vectorDeployment")
		return nil, err
	}

	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("No matching vectorDeployment found. Creating a new one...")

		// create new vectorDeployment
		vectorDeployment, err = constructVectorDeployment(r, stageVersion)
		if err != nil {
			log.Error(err, "Unable to construct vectorDeployment from template")
			return nil, err
		}

		if err := r.Create(ctx, vectorDeployment); err != nil {
			log.Error(err, "Unable to create vectorDeployment", "vectorDeployment", vectorDeployment)
			return nil, err
		}

		log.V(1).Info("Created vectorDeployment", "vectorDeployment", vectorDeployment)
	}

	return vectorDeployment, nil
}

func (r *StageVersionReconciler) getOrCreateVectorMigration(ctx context.Context, stageVersion *landscape.StageVersion) (*landscape.VectorMigration, error) {
	log := logf.FromContext(ctx)
	vectorMigration := &landscape.VectorMigration{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: stageVersion.Namespace,
		Name:      getVectorMigrationName(stageVersion),
	}, vectorMigration)

	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Unable to fetch vectorMigration")
		return nil, err
	}

	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("No matching vectorMigration found. Creating a new one...")

		// create new vectorMigration
		vectorMigration, err = constructVectorMigration(r, stageVersion)
		if err != nil {
			log.Error(err, "Unable to construct vectorMigration from template")
			return nil, err
		}

		if err := r.Create(ctx, vectorMigration); err != nil {
			log.Error(err, "Unable to create vectorMigration", "vectorMigration", vectorMigration)
			return nil, err
		}

		log.V(1).Info("Created vectorMigration", "vectorMigration", vectorMigration)
	}

	return vectorMigration, nil
}

func (r *StageVersionReconciler) getOrCreateVectorActivation(ctx context.Context, stageVersion *landscape.StageVersion) (*landscape.VectorActivation, error) {
	log := logf.FromContext(ctx)
	vectorActivation := &landscape.VectorActivation{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: stageVersion.Namespace,
		Name:      getVectorActivationName(stageVersion),
	}, vectorActivation)

	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Unable to fetch vectorActivation")
		return nil, err
	}

	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("No matching vectorActivation found. Creating a new one...")

		// create new vectorActivation
		vectorActivation, err = constructVectorActivation(r, stageVersion)
		if err != nil {
			log.Error(err, "Unable to construct vectorActivation from template")
			return nil, err
		}

		if err := r.Create(ctx, vectorActivation); err != nil {
			log.Error(err, "Unable to create vectorActivation", "vectorActivation", vectorActivation)
			return nil, err
		}

		log.V(1).Info("Created vectorActivation", "vectorActivation", vectorActivation)
	}

	return vectorActivation, nil
}

func (r *StageVersionReconciler) setStageVersionStatusTrue(ctx context.Context, req ctrl.Request, status string, message string) error {
	log := logf.FromContext(ctx)
	stageVersion := &landscape.StageVersion{}
	if err := r.Get(ctx, req.NamespacedName, stageVersion); err != nil {
		return err
	}

	if !meta.IsStatusConditionTrue(stageVersion.Status.Conditions, status) {
		meta.SetStatusCondition(&stageVersion.Status.Conditions, metav1.Condition{Type: status, Status: metav1.ConditionTrue, Reason: status, Message: message})
		if err := r.Status().Update(ctx, stageVersion); err != nil {
			log.Error(err, "Failed to update stageVersion status")
			return err
		}
	}

	return nil
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

func constructVectorActivation(r *StageVersionReconciler, stageVersion *landscape.StageVersion) (*landscape.VectorActivation, error) {
	vectorActivation := &landscape.VectorActivation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getVectorActivationName(stageVersion),
			Namespace: stageVersion.Namespace,
		},
		Spec: landscape.VectorActivationSpec{},
	}

	// set stageVersion as controller
	if err := controllerutil.SetControllerReference(stageVersion, vectorActivation, r.Scheme); err != nil {
		return nil, err
	}
	return vectorActivation, nil
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

func getVectorActivationName(stageVersion *landscape.StageVersion) string {
	return fmt.Sprintf("%s-%s", stageVersion.Name, "activation")
}
