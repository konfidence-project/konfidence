/*
Copyright 2026.

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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/konfidence-project/crds/api/global/v1alpha1"

	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller/domain"
)

const (
	defaultReconcileInterval = time.Minute
)

// VectorTemplateReconciler reconciles a VectorTemplate object
type VectorTemplateReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	OcmAdapter domain.VectorOcmPort
}

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectortemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectortemplates/status,verbs=get;update;patch

// Reconcile the VectorTemplate object to detect a vector drift and act upon it.
func (r *VectorTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling VectorTemplate", "name", req.NamespacedName)

	var vectorTemplate v1alpha1.VectorTemplate
	if err := r.Get(ctx, req.NamespacedName, &vectorTemplate); err != nil {
		log.Error(err, "unable to fetch VectorTemplate")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if err := r.detectAndActOnDrift(ctx, vectorTemplate); err != nil {
		log.Error(err, "error detecting or acting on drift for Vector template")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueAfterFromSpecOrDefault(vectorTemplate)}, nil
}

func (r *VectorTemplateReconciler) detectAndActOnDrift(ctx context.Context, template v1alpha1.VectorTemplate) error {
	log := logf.FromContext(ctx)

	desiredOcmComponents, err := mapComponentsToOCMReferences(template.Spec.Components)
	if err != nil {
		return fmt.Errorf("unable to map vector template components to ocm references: %w", err)
	}

	vectorOCMComponent, err := domain.NewOcmReference(template.Spec.UploadTarget)
	if err != nil {
		return fmt.Errorf("unable to create ocm reference from vector template upload target (%s): %w",
			template.Spec.UploadTarget, err)
	}

	desiredArtifacts, err := r.OcmAdapter.GetLatestArtifactVersions(ctx, desiredOcmComponents)
	if err != nil {
		return fmt.Errorf("unable to get desired artifacts for vector (%s): %w", vectorOCMComponent, err)
	}

	actualArtifacts, err := r.OcmAdapter.GetLatestArtifactsFromVector(ctx, vectorOCMComponent)
	if err != nil {
		return fmt.Errorf("unable to get actual artifacts from vector (%s): %w", vectorOCMComponent, err)
	}

	driftDetected := domain.HasDrift(desiredArtifacts, actualArtifacts)
	if !driftDetected {
		log.Info("No drift detected for vector - nothing to do")
		return nil
	}

	newVector := domain.Vector{
		Reference:     vectorOCMComponent,
		BaseReference: nil, // Set this later when implementing base support
		Artifacts:     desiredArtifacts,
	}
	err = r.OcmAdapter.CreateVector(ctx, newVector)
	if err != nil {
		return fmt.Errorf("unable to create new vector (%s) on drift: %w", vectorOCMComponent, err)
	}

	log.Info("Drift detected and new vector created successfully")
	return nil
}

func mapComponentsToOCMReferences(components []v1alpha1.Component) ([]domain.OcmReference, error) {
	ocmRefs := make([]domain.OcmReference, 0, len(components))
	for _, component := range components {
		componentOcmRef, err := domain.NewOcmReference(component.Name)
		if err != nil {
			return nil, fmt.Errorf("unable to create ocm reference from vector template component (%s): %w",
				component.Name, err)
		}
		ocmRefs = append(ocmRefs, componentOcmRef)
	}
	return ocmRefs, nil
}

func requeueAfterFromSpecOrDefault(vectorTemplate v1alpha1.VectorTemplate) time.Duration {
	if vectorTemplate.Spec.ReconcileInterval != nil {
		return vectorTemplate.Spec.ReconcileInterval.Duration
	}
	return defaultReconcileInterval
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.VectorTemplate{}).
		Named("vectortemplate").
		Complete(r)
}
