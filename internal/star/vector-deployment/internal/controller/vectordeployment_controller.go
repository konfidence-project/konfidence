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
	"crypto/sha256"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/konfidence-project/landscape-vector-deployment-controller/internal/controller/domain"
)

// VectorDeploymentReconciler reconciles a VectorDeployment object
type VectorDeploymentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	OcmAdapter domain.VectorOcmPort
}

// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=artifactdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectorassignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VectorDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *VectorDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)
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

	vector, err := mapVectorDeploymentToDomain(*vectorDeployment)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to map vector deployment %s to domain: %w", vectorDeployment.Name, err)
	}

	// if vector.ComponentSpec is empty then fetch Vector from OCI Repository
	if vector.ComponentSpec == "" {
		// 1. Fetch vector from OCI repository
		fetchedVector, err := r.OcmAdapter.GetVectorByReference(ctx, vectorDeployment.Namespace, vector.Reference)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to fetch vector OCM for vector deployment %s : %w", vectorDeployment.Name, err)
		}
		// 2. update vector.ComponentSpec and vector.Artifacts
		vector.ComponentSpec = fetchedVector.ComponentSpec
		vector.Artifacts = fetchedVector.Artifacts

		// 3. update vd.Status.ResolvedVectorOcm with json marshalled ComponentSpec
		vectorDeployment.Status.ResolvedVectorOcm = fetchedVector.ComponentSpec

		// 4. update status condition VectorDownloadedCondition to True
		meta.SetStatusCondition(
			&vectorDeployment.Status.Conditions,
			metav1.Condition{
				Type:   landscape.VectorDownloadedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDownloadedCondition,
				Message: fmt.Sprintf("Successfully downloaded vector %s from OCM repository %s", fetchedVector.Reference.Component, fetchedVector.Reference.OciRegistryUrl),
			},
		)
	}

	allDeploymentsReady, err := r.handleArtifactDeployments(ctx, *vector, vectorDeployment, log)
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
		return ctrl.Result{}, err
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
		return ctrl.Result{}, err
	}
	if !allAssignmentsReady {
		log.Info("waiting for vector assignments to be ready")
		return ctrl.Result{}, nil
	}

	// set status condition VectorReadyCondition to True
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.VectorReadyCondition,
		Status: metav1.ConditionTrue, Reason: landscape.VectorReadyCondition,
		Message: fmt.Sprintf("Vector deployment %s is ready", vectorDeployment.Name)})

	if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update vectorDeployment status: %w", patchError)
		}
	}

	log.Info("VectorDeployment reconciled successfully")
	return ctrl.Result{}, nil
}

// todo: remove vd *landscape.VectorDeployment from function signature
func (r *VectorDeploymentReconciler) handleArtifactDeployments(ctx context.Context, vector domain.Vector, vectorDeployment *landscape.VectorDeployment, log logr.Logger) (bool, error) {
	vectorDeployment.Status.ResultingArtifactDeployments = make(map[string]landscape.LocalArtifactDeploymentReference)
	vectorDeployment.Status.DeploymentResults = make(map[string]landscape.DeploymentResult)
	allReady := true

	// TODO parallelize and handle partial failures
	for _, artifact := range vector.Artifacts {
		// fetch the artifact component version from OCI
		artifactManifest, err := r.OcmAdapter.GetArtifactManifestByReference(ctx, vectorDeployment.Namespace, vector.Reference.OciRegistryUrl, artifact)
		if err != nil {
			return false, fmt.Errorf("failed to fetch artifact component version %q from repository %q: %w", artifact.ComponentName, vector.Reference.OciRegistryUrl, err)
		}

		var deploymentName string
		if artifactManifest.AllowReuse {
			deploymentName = constructArtifactDeploymentName(artifact.ComponentName, artifact.Version, nil)
		} else {
			uid := string(vectorDeployment.UID)
			deploymentName = constructArtifactDeploymentName(artifact.ComponentName, artifact.Version, &uid)
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
			artifactDeployment = r.constructArtifactDeployment(artifactManifest, vectorDeployment, deploymentName)
			if err := r.Create(ctx, artifactDeployment); err != nil {
				return false, fmt.Errorf("failed to ArtifactDeployment resource %s: %w", deploymentName, err)
			}

			log.Info("Created ArtifactDeployment", "name", deploymentName)
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
		} else {
			log.Info("ArtifactDeployment already has owner reference", "vector", vectorDeployment.Spec.Vector, "name", artifactDeployment.Name)
		}

		// Update the artifact deployment to the status map of the VectorDeployment
		vectorDeployment.Status.ResultingArtifactDeployments[artifact.ComponentName] = landscape.LocalArtifactDeploymentReference{
			Name: artifactDeployment.Name,
		}

		// state management for VectorDeployedCondition
		if meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, landscape.DeploymentResultCreatedCondition) {
			// collect deployment results
			for _, result := range artifactDeployment.Status.DeploymentResults {
				vectorDeployment.Status.DeploymentResults[artifact.ComponentName+"/"+result.Name] = result
			}
		}
		if !meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, landscape.ArtifactDeploymentReadyCondition) {
			allReady = false
		}
	}

	// set status condition ArtifactDeploymentsCreatedCondition to created
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.ArtifactDeploymentsCreatedCondition,
		Status: metav1.ConditionTrue, Reason: landscape.ArtifactDeploymentsCreatedCondition,
		Message: fmt.Sprintf("Successfully created Artifact deployments for vector deployment %s", vectorDeployment.Name)})

	if allReady {
		meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.VectorDeployedCondition,
			Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
			Message: fmt.Sprintf("All artifacts of vector deployment %s are deployed", vectorDeployment.Name)})
	}

	return allReady, nil
}

func (r *VectorDeploymentReconciler) handleVectorAssignments(ctx context.Context, vectorDeployment *landscape.VectorDeployment, log logr.Logger) (bool, error) {
	vectorDeployment.Status.ResultingVectorAssignments = make(map[string]landscape.LocalVectorAssignmentReference)
	allReady := true

	for componentName, artifactDeployment := range vectorDeployment.Status.ResultingArtifactDeployments {
		// fetch existing artifact assignment from k8s api
		vectorAssignment := &landscape.VectorAssignment{}
		assignmentName := fmt.Sprintf("assignment-%s-%s", artifactDeployment.Name, vectorDeployment.Name)
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

			log.Info("Created VectorAssignment", "name", assignmentName)
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
		} else {
			log.Info("Vector deployment already has owner reference", "vector", vectorDeployment.Spec.Vector, "name", vectorAssignment.Name)
		}

		// Update the artifact assignment to the status map of the VectorDeployment
		vectorDeployment.Status.ResultingVectorAssignments[componentName] = landscape.LocalVectorAssignmentReference{
			Name: vectorAssignment.Name,
		}

		// state management for VectorAssignmentsCreatedCondition
		if !meta.IsStatusConditionTrue(vectorAssignment.Status.Conditions, landscape.VectorAssignedCondition) {
			allReady = false
		}
	}

	// set status condition ArtifactDeploymentsCreatedCondition to created
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.VectorAssignmentsCreatedCondition,
		Status: metav1.ConditionTrue, Reason: landscape.VectorAssignmentsCreatedCondition,
		Message: fmt.Sprintf("Successfully created vector assignments for vector deployment %s", vectorDeployment.Name)})

	if allReady {
		meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: landscape.VectorReadyCondition,
			Status: metav1.ConditionTrue, Reason: landscape.VectorReadyCondition,
			Message: fmt.Sprintf("Vector deployment %s fully deployed", vectorDeployment.Name)})
	}

	return allReady, nil
}

func mapTaskManifestsToLandscape(taskManifests []domain.TaskManifest) []landscape.TaskManifest {
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

func mapArtifactResourcesToLandscape(resources []domain.OCMResource) []landscape.OCMResource {
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

func constructArtifactDeploymentName(artifactName, artifactVersion string, uid *string) string {
	h := sha256.New()
	h.Write([]byte(artifactName))
	h.Write([]byte(artifactVersion))

	if uid != nil {
		// makes the name unique to this vector deployment -> no reuse
		h.Write([]byte(*uid))
	}

	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash)
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

func (r *VectorDeploymentReconciler) constructArtifactDeployment(artifactManifest *domain.ArtifactManifest, vectorDeployment *landscape.VectorDeployment, deploymentName string) *landscape.ArtifactDeployment {
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
				Name:      artifactManifest.Type,
				Version:   artifactManifest.Version,
				Resources: artifactResources,
			},
		},
	}
}
