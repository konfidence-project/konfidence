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

	"github.com/go-logr/logr"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	pkgCtrl "github.com/konfidence-project/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	VectorDeploymentControllerName = "vector-deployment-controller"
	MaxLabelSize                   = 63
)

// VectorDeploymentReconciler reconciles a VectorDeployment object
type VectorDeploymentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   events.EventRecorder
	OcmAdapter VectorOcmPort
}

// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=artifactdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectorassignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *VectorDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling VectorDeployment")

	// get vector deployment usage
	vectorDeployment := &landscape.VectorDeployment{}
	if err := r.Get(ctx, req.NamespacedName, vectorDeployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Found vector deployment")

	originalVectorDeployment := vectorDeployment.DeepCopy()
	patch := client.MergeFrom(originalVectorDeployment)

	vectorRef, err := compref.Parse(vectorDeployment.Spec.Vector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse vector reference %s: %w", vectorDeployment.Spec.Vector, err)
	}

	var artifactRefs []compref.Ref

	// if ResolvedVectorOcm is empty then fetch Vector from OCI Repository
	if vectorDeployment.Status.ResolvedVectorOcm == "" {
		// 1. Fetch vector from OCI repository
		fetchedVectorDescriptor, err := r.OcmAdapter.GetVectorDescriptor(ctx, *vectorRef)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to fetch vector OCM for vector deployment %s : %w", vectorDeployment.Name, err)
		}

		// 2. persist descriptor JSON in status and extract artifact refs
		vectorDeployment.Status.ResolvedVectorOcm = string(fetchedVectorDescriptor.DescriptorJSON)
		artifactRefs = fetchedVectorDescriptor.References

		// 3. update status condition VectorDownloadedCondition to True
		meta.SetStatusCondition(
			&vectorDeployment.Status.Conditions,
			metav1.Condition{
				Type:               landscape.VectorDownloadedCondition,
				Status:             metav1.ConditionTrue,
				Reason:             landscape.VectorDownloadedCondition,
				Message:            fmt.Sprintf("Successfully downloaded vector %s from OCM repository", vectorDeployment.Spec.Vector),
				ObservedGeneration: vectorDeployment.Generation,
				LastTransitionTime: metav1.Now(),
			},
		)
	} else {
		// parse artifact refs from cached status
		artifactRefs, err = artifactRefsFromStatus(vectorDeployment.Status.ResolvedVectorOcm, *vectorRef)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to parse artifact refs from status for vector deployment %s: %w", vectorDeployment.Name, err)
		}
	}

	allDeploymentsReady, err := r.handleArtifactDeployments(ctx, artifactRefs, vectorDeployment, log)
	if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
			patchErrorMessage := "unable to update vectorDeployment status"

			if err != nil {
				reconcileError := fmt.Errorf("failed to handle artifact deployments for vector deployment %s : %w", vectorDeployment.Name, err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to handle artifact deployments for vector deployment %s: %w", vectorDeployment.Name, err)
	}
	if !allDeploymentsReady {
		log.Info("waiting for artifact deployments to be ready")
		return ctrl.Result{}, nil
	}

	allAssignmentsReady, err := r.handleVectorAssignments(ctx, vectorDeployment, log)
	if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
			patchErrorMessage := "unable to update vectorDeployment status"

			if err != nil {
				reconcileError := fmt.Errorf("failed to handle vector assignments for vector deployment %s : %w", vectorDeployment.Name, err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to handle vector assignments for vector deployment %s: %w", vectorDeployment.Name, err)
	}
	if !allAssignmentsReady {
		log.Info("waiting for vector assignments to be ready")
		return ctrl.Result{}, nil
	}

	// set status condition VectorReadyCondition to True
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               landscape.VectorReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             landscape.VectorReadyCondition,
		Message:            fmt.Sprintf("Vector deployment %s is ready", vectorDeployment.Name),
		ObservedGeneration: vectorDeployment.Generation,
		LastTransitionTime: metav1.Now(),
	})

	if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update vectorDeployment status: %w", patchError)
		}
	}

	log.Info("VectorDeployment reconciled successfully")
	return ctrl.Result{}, nil
}

func (r *VectorDeploymentReconciler) handleArtifactDeployments(ctx context.Context, artifactReferences []compref.Ref, vectorDeployment *landscape.VectorDeployment, log logr.Logger) (bool, error) {
	// Build fresh maps from scratch so removed artifacts are no longer referenced.
	// We use nil initially and allocate lazily to avoid spurious status patches when
	// DeepEqual compares nil (server value after omitempty round-trip) vs. empty map.
	var (
		resultingArtifactDeployments = make(map[string]landscape.LocalArtifactDeploymentReference, len(artifactReferences))
		deploymentResults            = make(map[string]landscape.DeploymentResult)
	)
	allReady := true

	// TODO parallelize and handle partial failures
	for _, artifactRef := range artifactReferences {
		// fetch the artifact component version from OCI
		artifactManifest, err := r.OcmAdapter.GetArtifactManifestByReference(ctx, artifactRef)
		if err != nil {
			return false, fmt.Errorf("failed to fetch artifact component version for %q: %w", artifactRef.String(), err)
		}

		var uid *string
		if !artifactManifest.AllowReuse {
			uid = new(string(vectorDeployment.UID))
		}

		deploymentName, err := ConstructArtifactDeploymentName(artifactRef.Component, artifactRef.Version, uid)
		if err != nil {
			return false, fmt.Errorf("failed to construct artifact deployment name for %q: %w", artifactRef.String(), err)
		}

		// fetch existing artifact deployment from k8s api
		artifactDeployment := &landscape.ArtifactDeployment{}
		err = r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: deploymentName}, artifactDeployment)
		if err != nil {
			// if error is not NotFound then return error
			if !apierrors.IsNotFound(err) {
				return false, fmt.Errorf("failed to get artifact deployment %q: %w", deploymentName, err)
			}

			log.Info("ArtifactDeployment not found, create new one", "name", deploymentName)
			artifactDeployment = r.constructArtifactDeployment(artifactRef, artifactManifest, vectorDeployment, deploymentName)
			if err := r.Create(ctx, artifactDeployment); err != nil {
				return false, fmt.Errorf("failed to create ArtifactDeployment resource %s: %w", deploymentName, err)
			}
			msg := fmt.Sprintf("Created ArtifactDeployment %s for VectorDeployment %s", deploymentName, vectorDeployment.Name)
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal, "ArtifactDeploymentCreated", "ArtifactDeploymentCreated", msg)
			log.Info(msg)
		} else {
			log.Info("ArtifactDeployment found, update existing one", "name", deploymentName)
		}

		var ownerRef *metav1.OwnerReference = nil
		for _, ref := range artifactDeployment.OwnerReferences {
			if ref.UID == vectorDeployment.UID {
				ownerRef = &ref
				break
			}
		}

		if ownerRef == nil {
			log.Info("Adding owner reference to existing artifact deployment", "vector", vectorDeployment.Spec.Vector, "name", artifactDeployment.Name)
			if err := controllerutil.SetOwnerReference(vectorDeployment, artifactDeployment, r.Scheme); err != nil {
				return false, fmt.Errorf("unable to add vectorDeployment owner reference to artifactDeployment: %w", err)
			}

			if err := r.Update(ctx, artifactDeployment); err != nil {
				return false, fmt.Errorf("failed to set owner reference for ArtifactDeployment %q: %w", artifactDeployment.Name, err)
			}
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal, "ArtifactDeploymentUpdated", "ArtifactDeploymentUpdated", fmt.Sprintf("Updated owner reference for ArtifactDeployment %s", artifactDeployment.Name))
		} else {
			log.Info("ArtifactDeployment already has owner reference", "vector", vectorDeployment.Spec.Vector, "name", artifactDeployment.Name)
		}

		// Update the artifact deployment to the status map of the VectorDeployment
		resultingArtifactDeployments[artifactRef.Component] = landscape.LocalArtifactDeploymentReference{
			Name: artifactDeployment.Name,
		}

		// state management for VectorDeployedCondition
		if meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, landscape.DeploymentResultCreatedCondition) {
			// collect deployment results
			for _, result := range artifactDeployment.Status.DeploymentResults {
				deploymentResults[artifactRef.Component+"/"+result.Name] = result
			}
		}
		if !meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, landscape.ArtifactDeploymentReadyCondition) {
			allReady = false
		}
	}

	// Write local maps back to status. Assign nil when empty so that the
	// omitempty JSON tag round-trips cleanly and reflect.DeepEqual stays stable.
	if len(resultingArtifactDeployments) > 0 {
		vectorDeployment.Status.ResultingArtifactDeployments = resultingArtifactDeployments
	} else {
		vectorDeployment.Status.ResultingArtifactDeployments = nil
	}
	if len(deploymentResults) > 0 {
		vectorDeployment.Status.DeploymentResults = deploymentResults
	} else {
		vectorDeployment.Status.DeploymentResults = nil
	}

	// set status condition ArtifactDeploymentsCreatedCondition to created
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               landscape.ArtifactDeploymentsCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             landscape.ArtifactDeploymentsCreatedCondition,
		Message:            fmt.Sprintf("Successfully created Artifact deployments for vector deployment %s", vectorDeployment.Name),
		ObservedGeneration: vectorDeployment.Generation,
		LastTransitionTime: metav1.Now(),
	})

	if allReady {
		meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
			Type:               landscape.VectorDeployedCondition,
			Status:             metav1.ConditionTrue,
			Reason:             landscape.VectorDeployedCondition,
			Message:            fmt.Sprintf("All artifacts of vector deployment %s are deployed", vectorDeployment.Name),
			ObservedGeneration: vectorDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})
	}

	return allReady, nil
}

func (r *VectorDeploymentReconciler) handleVectorAssignments(ctx context.Context, vectorDeployment *landscape.VectorDeployment, log logr.Logger) (bool, error) {
	resultingVectorAssignments := make(map[string]landscape.LocalVectorAssignmentReference, len(vectorDeployment.Status.ResultingArtifactDeployments))
	allReady := true

	for componentName, artifactDeployment := range vectorDeployment.Status.ResultingArtifactDeployments {
		// fetch existing artifact assignment from k8s api
		vectorAssignment := &landscape.VectorAssignment{}
		assignmentName := vectorDeployment.Name
		err := r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: assignmentName}, vectorAssignment)
		if err != nil {
			// if error is not NotFound then return error
			if !apierrors.IsNotFound(err) {
				return false, fmt.Errorf("failed to get vector assignment %q: %w", assignmentName, err)
			}

			// create a new VectorAssignment
			ad := &landscape.ArtifactDeployment{}
			err = r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: artifactDeployment.Name}, ad)
			if err != nil {
				return false, fmt.Errorf("failed to get artifact deployment %q for vector assignment %q: %w", artifactDeployment.Name, assignmentName, err)
			}

			log.Info("VectorAssignment not found, create new one", "name", assignmentName)

			vectorAssignment = &landscape.VectorAssignment{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      assignmentName,
					Namespace: vectorDeployment.Namespace,
					Labels: map[string]string{
						pkgCtrl.ArtifactReferenceLabel: artifactDeployment.Name,
					},
				},
				Spec: landscape.VectorAssignmentSpec{
					Manifest:              ad.Spec.Manifest,
					ArtifactDeploymentRef: artifactDeployment,
					VectorDeploymentRef: landscape.LocalVectorDeploymentReference{
						Name: vectorDeployment.Name,
					},
				},
			}
			if err := r.Create(ctx, vectorAssignment); err != nil {
				return false, fmt.Errorf("failed to create VectorAssignment %q: %w", assignmentName, err)
			}
			msg := fmt.Sprintf("Created VectorAssignment %s", assignmentName)
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal, "VectorAssignmentCreated", "VectorAssignmentCreated", msg)
			log.Info(msg)
		} else {
			log.Info("VectorAssignment found, update existing one", "name", assignmentName)
		}

		var ownerRef *metav1.OwnerReference = nil
		for _, ref := range vectorAssignment.OwnerReferences {
			if ref.UID == vectorDeployment.UID {
				ownerRef = &ref
				break
			}
		}

		if ownerRef == nil {
			log.Info("Adding owner reference to existing vector assignment", "vector", vectorDeployment.Spec.Vector, "name", vectorAssignment.Name)
			if err := controllerutil.SetControllerReference(vectorDeployment, vectorAssignment, r.Scheme); err != nil {
				return false, fmt.Errorf("unable to add vectorDeployment owner reference to vectorAssignment: %w", err)
			}

			if err := r.Update(ctx, vectorAssignment); err != nil {
				return false, fmt.Errorf("failed to set owner reference for VectorAssignment %q: %w", vectorAssignment.Name, err)
			}
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal, "VectorAssignmentUpdated", "VectorAssignmentUpdated", fmt.Sprintf("Updated owner reference for VectorAssignment %s", vectorAssignment.Name))
		} else {
			log.Info("Vector deployment already has owner reference", "vector", vectorDeployment.Spec.Vector, "name", vectorAssignment.Name)
		}

		// Update the artifact assignment to the status map of the VectorDeployment
		resultingVectorAssignments[componentName] = landscape.LocalVectorAssignmentReference{
			Name: vectorAssignment.Name,
		}

		// state management for VectorAssignmentsCreatedCondition
		if !meta.IsStatusConditionTrue(vectorAssignment.Status.Conditions, landscape.VectorAssignmentReadyCondition) {
			allReady = false
		}
	}

	// Write local map back to status. Assign nil when empty so that the
	// omitempty JSON tag round-trips cleanly and reflect.DeepEqual stays stable.
	if len(resultingVectorAssignments) > 0 {
		vectorDeployment.Status.ResultingVectorAssignments = resultingVectorAssignments
	} else {
		vectorDeployment.Status.ResultingVectorAssignments = nil
	}

	// set status condition ArtifactDeploymentsCreatedCondition to created
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               landscape.VectorAssignmentsCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             landscape.VectorAssignmentsCreatedCondition,
		Message:            fmt.Sprintf("Successfully created vector assignments for vector deployment %s", vectorDeployment.Name),
		ObservedGeneration: vectorDeployment.Generation,
		LastTransitionTime: metav1.Now(),
	})

	if allReady {
		meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
			Type:               landscape.VectorReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             landscape.VectorReadyCondition,
			Message:            fmt.Sprintf("Vector deployment %s fully deployed", vectorDeployment.Name),
			ObservedGeneration: vectorDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})
	}

	return allReady, nil
}

func mapTaskManifestsToLandscape(taskManifests []TaskManifest) []landscape.TaskManifest {
	landscapeTaskManifests := make([]landscape.TaskManifest, len(taskManifests))
	for i, taskManifest := range taskManifests {
		landscapeTaskManifests[i] = landscape.TaskManifest{
			Name:      taskManifest.Name,
			Type:      taskManifest.Type,
			DependsOn: taskManifest.DependsOn,
			Spec:      runtime.RawExtension{Raw: []byte(taskManifest.Spec)},
		}
	}
	return landscapeTaskManifests
}

func mapArtifactResourcesToLandscape(resources []OCMResource) []landscape.OCMResource {
	landscapeResources := make([]landscape.OCMResource, 0, len(resources))
	for _, resource := range resources {
		landscapeResources = append(landscapeResources, landscape.OCMResource{
			Name:    resource.Name,
			Content: runtime.RawExtension{Raw: resource.Content},
			Type:    resource.Type,
		})
	}
	return landscapeResources
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorDeploymentReconciler) SetupWithManager(mgr ctrl.Manager, controllerName string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorDeployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&landscape.ArtifactDeployment{}, builder.MatchEveryOwner).
		Owns(&landscape.VectorAssignment{}, builder.MatchEveryOwner).
		Named(controllerName).
		Complete(r)
}

func (r *VectorDeploymentReconciler) constructArtifactDeployment(ref compref.Ref, artifactManifest ArtifactManifest, vectorDeployment *landscape.VectorDeployment, deploymentName string) *landscape.ArtifactDeployment {
	// map task manifests from domain.TaskManifest to landscape.TaskManifest
	taskManifests := mapTaskManifestsToLandscape(artifactManifest.Tasks)
	artifactResources := mapArtifactResourcesToLandscape(artifactManifest.Resources)
	return &landscape.ArtifactDeployment{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      deploymentName,
			Namespace: vectorDeployment.Namespace,
		},
		Spec: landscape.ArtifactDeploymentSpec{
			Manifest: landscape.ArtifactManifest{
				Type:       artifactManifest.Type,
				AllowReuse: artifactManifest.AllowReuse,
			},
			TaskManifests: taskManifests,
			Component: landscape.OCMComponent{
				Name:      ref.Component,
				Version:   ref.Version,
				Resources: artifactResources,
			},
		},
	}
}
