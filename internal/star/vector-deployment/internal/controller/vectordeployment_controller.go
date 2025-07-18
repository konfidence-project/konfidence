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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
)

// VectorDeploymentReconciler reconciles a VectorDeployment object
type VectorDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=vectordeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=artifactdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=vectorassignments,verbs=get;list;watch;create;update;patch;delete

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
		log.Error(err, "Unable to fetch vector deployment")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Found vector deployment")
	artifacts, err := getArtifactsFromVector(vd.Spec.VectorRef)
	if err != nil {
		log.Error(err, "Failed to get artifacts from vector")
		return ctrl.Result{}, err
	}

	for _, artifact := range artifacts {
		name := constructArtifactDeploymentName(artifact) //TODO figure out naming when reuse is disabled
		found := &landscape.ArtifactDeployment{}
		if err = r.Get(ctx, types.NamespacedName{Namespace: vd.Namespace, Name: name}, found); client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get ArtifactDeployment", "name", name)
			return ctrl.Result{}, err
		}
		if found != nil && found.Spec.Manifest.AllowReuse {
			var ownerRef *metav1.OwnerReference = nil
			for _, ref := range found.OwnerReferences {
				if ref.UID == vd.UID {
					ownerRef = &ref
					break
				}
			}
			if ownerRef == nil {
				log.Info("Adding owner reference to existing artifact deployment", "vector", vd.Spec.VectorRef, "name", found.Name)
				found.OwnerReferences = append(found.OwnerReferences, constructVectorDeploymentOwnerReference(vd))
				if err := r.Update(ctx, found); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				log.Info("Vector deployment already has owner reference", "vector", vd.Spec.VectorRef, "name", found.Name)
			}

		} else {
			ad, err := constructArtifactDeployment(name, vd.Namespace, artifact)
			if err != nil {
				log.Error(err, "Failed to construct ArtifactDeployment", "name", name)
				return ctrl.Result{}, err
			}
			ad.OwnerReferences = append(ad.OwnerReferences, constructVectorDeploymentOwnerReference(vd))

			if err := r.Create(ctx, ad); err != nil {
				log.Error(err, "Failed to create ArtifactDeployment", "name", name)
				return ctrl.Result{}, err
			}
			log.Info("Created ArtifactDeployment", "name", name)
		}
	}

	return ctrl.Result{}, nil
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

func constructArtifactDeployment(name string, namespace string, artifact landscape.OCMComponent) (*landscape.ArtifactDeployment, error) {
	manifest, err := findArtifactManifestFromOCM(artifact)
	if err != nil {
		// Log the error and return nil or handle it as needed
		logf.Log.Error(err, "Failed to find artifact type for component", "component", artifact.Name)
		return nil, err
	}

	return &landscape.ArtifactDeployment{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.ArtifactDeploymentSpec{
			Manifest:  *manifest,
			Component: artifact,
		},
	}, nil
}

func findArtifactManifestFromOCM(artifact landscape.OCMComponent) (*landscape.ArtifactManifest, error) {
	var found *landscape.OCMResource = nil
	for _, r := range artifact.Resources {
		if r.Type == "cloud.konfidence.artifact.manifest" {
			if found != nil {
				return nil, fmt.Errorf("multiple artifact manifests found for component %s", artifact.Name)
			}
			found = &r
		}
	}

	if found == nil {
		return nil, fmt.Errorf("no artifact manifest found for component %s", artifact.Name)
	}

	// TODO ocm magic to get actual value
	return &landscape.ArtifactManifest{
		Type:       "cloud.konfidence.flux",
		AllowReuse: true,
	}, nil
}

func constructArtifactDeploymentName(artifact landscape.OCMComponent) string {
	h := sha256.New()
	h.Write([]byte(artifact.Name))
	h.Write([]byte(artifact.Version))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash)
}

func getArtifactsFromVector(ref v1.TypedLocalObjectReference) ([]landscape.OCMComponent, error) {
	return []landscape.OCMComponent{
		{
			Name:    "example-service-a",
			Version: "1.0.0",
			Resources: []landscape.OCMResource{
				{
					Name:    "konfidence-manifest",
					Type:    "cloud.konfidence.artifact.manifest",
					Image:   "todo",
					Version: "1.0.0",
				},
				{
					Name:    "example-resource",
					Type:    "example-type",
					Image:   "example-image",
					Version: "1.0.0",
				},
			},
		},
		{
			Name:    "example-service-b",
			Version: "1.0.0",
			Resources: []landscape.OCMResource{
				{
					Name:    "konfidence-manifest",
					Type:    "cloud.konfidence.artifact.manifest",
					Image:   "todo",
					Version: "1.0.0",
				},
				{
					Name:    "example-resource",
					Type:    "example-type",
					Image:   "example-image",
					Version: "1.0.0",
				},
			},
		},
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		Named("vectordeployment").
		For(&landscape.VectorDeployment{}).
		Owns(&landscape.ArtifactDeployment{}).
		Owns(&landscape.VectorAssignment{}).
		Complete(r)
}
