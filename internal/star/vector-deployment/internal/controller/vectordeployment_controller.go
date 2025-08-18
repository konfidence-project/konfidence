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
	"k8s.io/apimachinery/pkg/util/json"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/compdesc"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"ocm.software/ocm/api/ocm/extensions/repositories/ocireg"
	"ocm.software/ocm/api/tech/oci/identity"
)

// VectorDeploymentReconciler reconciles a VectorDeployment object
type VectorDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=artifactdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectorassignments,verbs=get;list;watch;create;update;patch;delete

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

	ocmRef, err := parseComponentVersionUrl(vd.Spec.Vector)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to parse vector reference %q", vd.Spec.Vector)
	}

	descriptor, err := r.handleVectorDownload(ctx, vd, ocmRef, log)
	if err != nil {
		log.Error(err, "Failed to handle vector download")
		return ctrl.Result{}, errors.Wrapf(err, "failed to handle vector download for vector deployment %s", vd.Name)
	}

	err = r.handleArtifactDeployments(ctx, vd, descriptor, ocmRef, log)
	if err != nil {
		log.Error(err, "Failed to handle artifact deployments")
		return ctrl.Result{}, errors.Wrapf(err, "failed to handle artifact deployments for vector deployment %s", vd.Name)
	}

	return ctrl.Result{}, nil
}

//TODO aggregate status of artifact deployments

func (r *VectorDeploymentReconciler) handleArtifactDeployments(ctx context.Context, vd *landscape.VectorDeployment, descriptor compdesc.ComponentSpec, ocmRef ocm.RefSpec, log logr.Logger) error {
	vd.Status.ResultingArtifactDeployments = make(map[string]corev1.TypedObjectReference)
	//TODO parallelize and handle partial failures
	for _, artifactRef := range descriptor.References {
		version := artifactRef.GetVersion()
		// create a new ocm reference for the artifact component version with the same repository as the vector
		artifactOcmRef := ocm.RefSpec{
			UniformRepositorySpec: ocmRef.UniformRepositorySpec,
			CompSpec: ocm.CompSpec{
				Component: artifactRef.GetComponentName(),
				Version:   &version,
			},
		}
		artifact, err := fetchOcm(artifactOcmRef)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch artifact component version %q from repository %q", artifactRef.GetComponentName(), ocmRef.UniformRepositorySpec.String())
		}

		name := constructArtifactDeploymentName(*artifact) //TODO figure out naming when reuse is disabled
		ad := landscape.ArtifactDeployment{}
		if err = r.Get(ctx, types.NamespacedName{Namespace: vd.Namespace, Name: name}, &ad); client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get ArtifactDeployment", "name", name)
			return err
		}
		if apierrors.IsNotFound(err) && ad.Spec.Manifest.AllowReuse {
			var ownerRef *metav1.OwnerReference = nil
			for _, ref := range ad.OwnerReferences {
				if ref.UID == vd.UID {
					ownerRef = &ref
					break
				}
			}
			if ownerRef == nil {
				log.Info("Adding owner reference to existing artifact deployment", "vector", vd.Spec.Vector, "name", ad.Name)
				ad.OwnerReferences = append(ad.OwnerReferences, constructVectorDeploymentOwnerReference(vd))
				if err := r.Update(ctx, &ad); err != nil {
					return err
				}
			} else {
				log.Info("Vector deployment already has owner reference", "vector", vd.Spec.Vector, "name", ad.Name)
			}

		} else {
			ad, err = constructArtifactDeployment(name, vd.Namespace, *artifact)
			if err != nil {
				log.Error(err, "Failed to construct ArtifactDeployment", "name", name)
				return err
			}
			ad.OwnerReferences = append(ad.OwnerReferences, constructVectorDeploymentOwnerReference(vd))

			if err := r.Create(ctx, &ad); err != nil {
				log.Error(err, "Failed to create ArtifactDeployment", "name", name)
				return err
			}
			log.Info("Created ArtifactDeployment", "name", name)
		}

		// Add the artifact deployment to the map
		vd.Status.ResultingArtifactDeployments[artifactRef.GetName()] = corev1.TypedObjectReference{
			APIGroup:  &ad.APIVersion, // FIXME: difference between APIGroup and APIVersion?
			Kind:      ad.Kind,
			Namespace: &ad.Namespace,
			Name:      ad.Name,
		}
	}

	meta.SetStatusCondition(&vd.Status.Conditions, metav1.Condition{Type: landscape.ArtifactDeploymentsCreatedCondition,
		Status: metav1.ConditionTrue, Reason: landscape.ArtifactDeploymentsCreatedCondition,
		Message: fmt.Sprintf("Successfully created Artifact deployments for vector deployment %s", vd.Name)})

	if err := r.Status().Update(ctx, vd); err != nil {
		log.Error(err, "Failed to update vector deployment status")
		return err
	}
	return nil
}

func (r *VectorDeploymentReconciler) handleVectorDownload(ctx context.Context, vd *landscape.VectorDeployment, ocmRef ocm.RefSpec, log logr.Logger) (compdesc.ComponentSpec, error) {
	var descriptor compdesc.ComponentSpec
	if vd.Status.ResolvedVectorOcm == "" {
		vectorOcm, err := fetchOcm(ocmRef)
		if err != nil {
			log.Error(err, "Failed to fetch vector OCM")
			return compdesc.ComponentSpec{}, err
		}

		descriptor = (*vectorOcm).GetDescriptor().ComponentSpec
		vectorOcmJson, err := json.Marshal(descriptor)
		if err != nil {
			return compdesc.ComponentSpec{}, err
		}
		vd.Status.ResolvedVectorOcm = string(vectorOcmJson)
		meta.SetStatusCondition(&vd.Status.Conditions, metav1.Condition{Type: landscape.VectorDownloadedCondition,
			Status: metav1.ConditionTrue, Reason: landscape.VectorDownloadedCondition,
			Message: fmt.Sprintf("Successfully downloaded vector %s from OCM repository %s", ocmRef.Component, ocmRef.UniformRepositorySpec.String())})

		if err := r.Status().Update(ctx, vd); err != nil {
			log.Error(err, "Failed to update vector deployment status")
			return compdesc.ComponentSpec{}, err
		}
	} else {
		err := json.Unmarshal([]byte(vd.Status.ResolvedVectorOcm), &descriptor)
		if err != nil {
			log.Error(err, "Failed to unmarshal resolved vector OCM")
			return compdesc.ComponentSpec{}, errors.Wrapf(err, "failed to unmarshal resolved vector OCM for vector deployment %s", vd.Name)
		}
	}
	return descriptor, nil
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

func constructArtifactDeployment(name string, namespace string, artifact ocm.ComponentVersionAccess) (landscape.ArtifactDeployment, error) {
	manifest, err := findArtifactManifestFromOCM(artifact)
	if err != nil {
		// Log the error and return nil or handle it as needed
		logf.Log.Error(err, "Failed to find artifact type for component", "component", artifact.GetName())
		return landscape.ArtifactDeployment{}, err
	}

	artifactSpec := landscape.OCMComponent{
		Name:     artifact.GetName(),
		Version:  artifact.GetVersion(),
		Provider: string(artifact.GetProvider().Name),
	}
	for _, ref := range artifact.GetResources() {
		artifactSpec.Resources = append(artifactSpec.Resources, landscape.OCMResource{
			Name:    ref.Meta().Name,
			Type:    ref.Meta().Type,
			Version: ref.Meta().Version,
			Image:   "some-image", //FIXME, should be replaced with actual image logic
		})
	}

	return landscape.ArtifactDeployment{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: landscape.ArtifactDeploymentSpec{
			Manifest:  *manifest,
			Component: artifactSpec,
		},
	}, nil
}

func findArtifactManifestFromOCM(artifact ocm.ComponentVersionAccess) (*landscape.ArtifactManifest, error) {
	var found *ocm.ResourceAccess = nil
	for _, r := range artifact.GetResources() {
		if r.Meta().Type == "cloud.konfidence.artifact.manifest" {
			if found != nil {
				return nil, fmt.Errorf("multiple artifact manifests found for component %s", artifact.GetName())
			}
			found = &r
		}
	}

	if found == nil {
		return nil, fmt.Errorf("no artifact manifest found for component %s", artifact.GetName())
	}

	// TODO ocm magic to get actual value
	return &landscape.ArtifactManifest{
		Type:       "cloud.konfidence.flux",
		AllowReuse: true,
	}, nil
}

func constructArtifactDeploymentName(artifact ocm.ComponentVersionAccess) string {
	h := sha256.New()
	h.Write([]byte(artifact.GetName()))
	h.Write([]byte(artifact.GetVersion()))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash)
}

func fetchOcm(ref ocm.RefSpec) (*ocm.ComponentVersionAccess, error) {
	ctx := ocm.DefaultContext()
	creds := identity.SimpleCredentials("d060274", "<some-token>")

	spec := ocireg.NewRepositorySpec(ref.UniformRepositorySpec.String())
	repo, err := ctx.RepositoryForSpec(spec, creds)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot setup repository")
	}
	defer repo.Close()

	vectorOcm, err := fetchOcmComponentVersionFromRepo(repo, ref.Component, *ref.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch component version %q from repository %q", ref.Component, ref.UniformRepositorySpec.String())
	}

	return vectorOcm, nil
}

func parseComponentVersionUrl(ref string) (ocm.RefSpec, error) {
	ocmRef, err := ocm.ParseRef(ref)
	if err != nil {
		return ocm.RefSpec{}, errors.Wrapf(err, "invalid vector reference %q", ref)
	}
	if !ocmRef.IsVersion() {
		return ocm.RefSpec{}, errors.Errorf("vector reference %q is not a version", ref)
	}
	return ocmRef, nil
}

func fetchOcmComponentVersionFromRepo(repo ocm.Repository, component string, version string) (*ocm.ComponentVersionAccess, error) {
	cv, err := repo.LookupComponentVersion(component, version)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot lookup component version")
	}

	return &cv, nil
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
