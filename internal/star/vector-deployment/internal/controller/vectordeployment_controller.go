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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

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
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

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
	vd := &landscape.VectorDeployment{}
	if err := r.Get(ctx, req.NamespacedName, vd); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Found vector deployment")

	vector, err := mapVectorDeploymentToDomain(*vd)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to map vector deployment %s to domain: %w", vd.Name, err)
	}

	// if vector.ComponentSpec is empty then fetch Vector from OCI Repository
	if vector.ComponentSpec == "" {
		// 1. Fetch vector from OCI repository
		fetchedVector, err := r.OcmAdapter.GetVectorByReference(ctx, vector.Reference)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to fetch vector OCM for vector deployment %s : %w", vd.Name, err)
		}
		// 2. update vector.ComponentSpec and vector.Artifacts
		vector.ComponentSpec = fetchedVector.ComponentSpec
		vector.Artifacts = fetchedVector.Artifacts

		// 3. update vd.Status.ResolvedVectorOcm with json marshalled ComponentSpec
		vd.Status.ResolvedVectorOcm = fetchedVector.ComponentSpec

		// 4. update status condition VectorDownloadedCondition to True
		meta.SetStatusCondition(
			&vd.Status.Conditions,
			metav1.Condition{
				Type:   landscape.VectorDownloadedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDownloadedCondition,
				Message: fmt.Sprintf("Successfully downloaded vector %s from OCM repository %s", fetchedVector.Reference.Component, fetchedVector.Reference.OciRegistryUrl),
			},
		)

		// 4. update status in k8s
		// todo: use patch instead of update to avoid conflicts and multiple retries
		if err := r.Status().Update(ctx, vd); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update VectorDeployment status of %s: %w", vd.Name, err)
		}
	}

	err = r.handleArtifactDeployments(ctx, *vector, vd, log)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to handle artifact deployments for vector deployment %s : %w", vd.Name, err)
	}

	return ctrl.Result{}, nil
}

// todo: remove vd *landscape.VectorDeployment from function signature
func (r *VectorDeploymentReconciler) handleArtifactDeployments(ctx context.Context, vector domain.Vector, vd *landscape.VectorDeployment, log logr.Logger) error {
	vd.Status.ResultingArtifactDeployments = make(map[string]corev1.TypedObjectReference)
	allReady := true

	// TODO parallelize and handle partial failures
	for _, artifact := range vector.Artifacts {
		// fetch the artifact component version from OCI
		artifactManifest, err := r.OcmAdapter.GetArtifactManifestByReference(ctx, vector.Reference.OciRegistryUrl, artifact)
		if err != nil {
			return fmt.Errorf("failed to fetch artifact component version %q from repository %q: %w", artifact.ComponentName, vector.Reference.OciRegistryUrl, err)
		}

		var deploymentName string
		if artifactManifest.AllowReuse {
			deploymentName = constructArtifactDeploymentName(artifact.ComponentName, artifact.Version, nil)
		} else {
			uid := string(vd.UID)
			deploymentName = constructArtifactDeploymentName(artifact.ComponentName, artifact.Version, &uid)
		}

		// fetch existing artifact deployment from k8s api
		artifactDeployment := landscape.ArtifactDeployment{}
		err = r.Get(ctx, types.NamespacedName{Namespace: vd.Namespace, Name: deploymentName}, &artifactDeployment)
		if err != nil {
			// if error is not NotFound then return error
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get artifact deployment %q: %w", deploymentName, err)
			}
			log.Info("ArtifactDeployment not found, create new one", "name", deploymentName)

			// map task manifests from domain.TaskManifest to landscape.TaskManifest
			taskManifests := mapTaskManifestsToLandscape(artifactManifest.Tasks)
			artifactDeployment.ObjectMeta = ctrl.ObjectMeta{
				Name:      deploymentName,
				Namespace: vd.Namespace,
			}
			artifactDeployment.Spec = landscape.ArtifactDeploymentSpec{
				Manifest: landscape.ArtifactManifest{
					Type:       artifactManifest.Type,
					AllowReuse: artifactManifest.AllowReuse,
				},
				TaskManifests:  taskManifests,
				ArtifactOcmRef: artifactManifest.OciRegistryUrl,
				ArtifactOcm:    artifactManifest.ComponentSpec,
			}

			if err := r.Create(ctx, &artifactDeployment); err != nil {
				return fmt.Errorf("failed to ArtifactDeployment resource %s: %w", deploymentName, err)
			}
			log.Info("Created ArtifactDeployment", "name", deploymentName)

		} else {
			log.Info("ArtifactDeployment found, update existing one", "name", deploymentName)
			if artifactManifest.AllowReuse {
				var ownerRef *metav1.OwnerReference = nil
				for _, ref := range artifactDeployment.OwnerReferences {
					if ref.UID == vd.UID {
						ownerRef = &ref
						break
					}
				}
				if ownerRef == nil {
					log.Info("Adding owner reference to existing artifact deployment", "vector", vd.Spec.Vector, "name", artifactDeployment.Name)
					artifactDeployment.OwnerReferences = append(artifactDeployment.OwnerReferences, constructVectorDeploymentOwnerReference(vd))
					if err := r.Update(ctx, &artifactDeployment); err != nil {
						return err
					}
				} else {
					log.Info("Vector deployment already has owner reference", "vector", vd.Spec.Vector, "name", artifactDeployment.Name)
				}
			}
		}
		// Update the artifact deployment to the status map of the VectorDeployment
		vd.Status.ResultingArtifactDeployments[artifact.ComponentName] = corev1.TypedObjectReference{
			APIGroup:  &artifactDeployment.APIVersion, // FIXME: difference between APIGroup and APIVersion?
			Kind:      artifactDeployment.Kind,
			Namespace: &artifactDeployment.Namespace,
			Name:      artifactDeployment.Name,
		}

		// state management for VectorDeployedCondition
		if meta.IsStatusConditionFalse(artifactDeployment.Status.Conditions, landscape.ArtifactDeployedCondition) {
			allReady = false
		}
	} // end for loop

	// set status condition ArtifactDeploymentsCreatedCondition to created
	meta.SetStatusCondition(&vd.Status.Conditions, metav1.Condition{Type: landscape.ArtifactDeploymentsCreatedCondition,
		Status: metav1.ConditionTrue, Reason: landscape.ArtifactDeploymentsCreatedCondition,
		Message: fmt.Sprintf("Successfully created Artifact deployments for vector deployment %s", vd.Name)})

	if allReady {
		meta.SetStatusCondition(&vd.Status.Conditions, metav1.Condition{Type: landscape.VectorDeployedCondition,
			Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
			Message: fmt.Sprintf("All artifacts of vector deployment %s are deployed", vd.Name)})
	}

	// update status in k8s
	// todo: use patch instead of update to avoid conflicts and multiple retries
	if err := r.Status().Update(ctx, vd); err != nil {
		return fmt.Errorf("failed to update VectorDeployment status of %s: %w", vd.Name, err)
	}

	return nil
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

// TODO factor out owner reference construction to a common place
func constructVectorDeploymentOwnerReference(vdu *landscape.VectorDeployment) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: vdu.APIVersion,
		Kind:       vdu.Kind,
		Name:       vdu.Name,
		UID:        vdu.UID,
	}
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
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		Named(controllerName).
		For(&landscape.VectorDeployment{}).
		Owns(&landscape.ArtifactDeployment{}, builder.MatchEveryOwner).
		Owns(&landscape.VectorAssignment{}, builder.MatchEveryOwner).
		Complete(r)
}
