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
	"errors"
	"fmt"
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/konfidence-project/crds/api/global/v1alpha1"

	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller/domain"
)

const (
	defaultReconcileInterval = time.Minute

	VectorAssemblyControllerName = "gcp-vector-assembly-controller"

	EventActionStatusPatch    = "StatusPatch"
	EventActionDriftDetection = "DriftDetection"
	EventActionVectorCreation = "VectorCreation"
)

// VectorTemplateReconciler reconciles a VectorTemplate object
type VectorTemplateReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	OcmAdapter domain.VectorOcmPort
	Recorder   events.EventRecorder // TODO: Use a EventRecorderLogger once sigs.k8s.io/controller-runtime provides an implementation.
}

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectortemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=vectortemplates/status,verbs=get;update;patch

// Reconcile the VectorTemplate object to detect a vector drift and act upon it.
func (r *VectorTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling VectorTemplate", "name", req.NamespacedName)

	var vectorTemplate v1alpha1.VectorTemplate
	if err := r.Get(ctx, req.NamespacedName, &vectorTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch VectorTemplate")
		return ctrl.Result{}, err
	}

	// preserve original vector template status for patching it later
	originalVectorTemplate := vectorTemplate.DeepCopy()
	patch := client.MergeFrom(originalVectorTemplate)

	var err error
	if err = r.detectAndActOnDrift(ctx, &vectorTemplate); err != nil {
		log.Error(err, "error detecting or acting on drift for Vector template")
	}

	// patch the vector template status updates (regardless of error in drift detection/handling)
	var patchErr error
	if !reflect.DeepEqual(vectorTemplate.Status, originalVectorTemplate.Status) {
		if patchErr = r.Client.Status().Patch(ctx, &vectorTemplate, patch); patchErr != nil {
			log.Error(patchErr, "unable to patch vector template status")
			r.Recorder.Eventf(&vectorTemplate, nil, v1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch, patchErr.Error())
		}
	}

	err = errors.Join(err, patchErr)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueAfterFromSpecOrDefault(vectorTemplate)}, nil
}

func (r *VectorTemplateReconciler) detectAndActOnDrift(ctx context.Context, template *v1alpha1.VectorTemplate) error {
	log := logf.FromContext(ctx)

	ocmComponentsFromComponentList, err := mapComponentsToOCMReferences(template.Spec.Components)
	if err != nil {
		err = fmt.Errorf("unable to map vector template components to ocm references: %w", err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             v1alpha1.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, v1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	vectorOCMComponent, err := domain.NewOcmReference(template.Spec.UploadTarget)
	if err != nil {
		err = fmt.Errorf("unable to create ocm reference from vector template upload target (%s): %w", template.Spec.UploadTarget, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             v1alpha1.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, v1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	var desiredArtifacts []domain.Artifact
	if template.Spec.Base != nil {
		if desiredArtifacts, err = r.getArtifactsFromBaseVector(ctx, template, vectorOCMComponent.Component); err != nil {
			return err
		}
	}

	latestArtifactsFromComponentList, err := r.OcmAdapter.GetLatestArtifactVersions(ctx, ocmComponentsFromComponentList)
	if err != nil {
		err = fmt.Errorf("unable to get desired artifacts for vector (%s): %w", vectorOCMComponent, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             v1alpha1.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, v1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	desiredArtifacts = combineBaseArtifactsAndComponentArtifacts(desiredArtifacts, latestArtifactsFromComponentList)

	actualVector, err := r.OcmAdapter.GetLatestVector(ctx, vectorOCMComponent)
	if errors.Is(err, domain.ErrVectorNotFound) {
		msg := "Vector not found in OCM repository - creating new vector"
		r.Recorder.Eventf(template, nil, v1.EventTypeNormal, "VectorNotFound", "ResolvingLatestVector", msg)
		log.Info(msg, "VectorOCMComponent", vectorOCMComponent.Component)
	} else if err != nil {
		err = fmt.Errorf("unable to get actual artifacts from vector (%s): %w", vectorOCMComponent, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             v1alpha1.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, v1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	driftDetected := domain.HasDrift(desiredArtifacts, actualVector.Artifacts)
	if !driftDetected {
		msg := fmt.Sprintf("No drift detected for vector - vector version is still %s", actualVector.Version)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.VectorTemplateReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             v1alpha1.VectorTemplateNoDriftDetectedReason,
			Message:            msg,
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, v1.EventTypeNormal, v1alpha1.VectorTemplateNoDriftDetectedReason, EventActionDriftDetection, msg)
		log.Info(msg, "VectorVersion", actualVector.Version, "VectorOCMComponent", vectorOCMComponent.Component)
		return nil
	}

	newVector := domain.Vector{
		// TODO: use https://github.com/open-component-model/ocm-spec/blob/main/doc/04-extensions/03-storage-backends/oci.md#121-version-aliasing
		Version:   time.Now().UTC().Format("2006.1.2-150405000Z"),
		Reference: vectorOCMComponent,
		Artifacts: desiredArtifacts,
	}
	err = r.OcmAdapter.CreateVector(ctx, newVector)
	if err != nil {
		err = fmt.Errorf("unable to create new vector (%s) on drift: %w", vectorOCMComponent, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.VectorTemplateReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             v1alpha1.VectorTemplateVectorCreationFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, v1.EventTypeWarning, v1alpha1.VectorTemplateVectorCreationFailedReason, EventActionVectorCreation, err.Error())
		return err
	}

	msg := fmt.Sprintf("Drift detected and new vector created successfully - new vector version is %s", newVector.Version)
	meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
		Type:               v1alpha1.VectorTemplateReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             v1alpha1.VectorTemplateVectorCreatedReason,
		Message:            msg,
		ObservedGeneration: template.Generation,
		LastTransitionTime: metav1.Now(),
	})
	r.Recorder.Eventf(template, nil, v1.EventTypeNormal, v1alpha1.VectorTemplateVectorCreatedReason, "VectorCreation", msg)
	log.Info(msg, "VectorVersion", newVector.Version, "VectorOCMComponent", vectorOCMComponent.Component)
	return nil
}

func (r *VectorTemplateReconciler) getArtifactsFromBaseVector(ctx context.Context, template *v1alpha1.VectorTemplate, vectorOCMComponentName string) ([]domain.Artifact, error) {
	baseVectorOCMComponent, err := domain.NewOcmReference(*template.Spec.Base)
	if err != nil {
		err = fmt.Errorf("unable to create ocm reference from vector template base (%s): %w", *template.Spec.Base, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             v1alpha1.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, v1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return nil, err
	}

	baseVector, err := r.OcmAdapter.GetLatestVector(ctx, baseVectorOCMComponent)
	if err != nil {
		err = fmt.Errorf("unable to get artifacts from base vector (%s): %w", baseVectorOCMComponent, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               v1alpha1.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             v1alpha1.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, v1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return nil, err
	}
	log := logf.FromContext(ctx)
	log.Info("Using base vector for vector OCM component", "BaseVectorVersion", baseVector.Version, "BaseVectorOCMComponent", baseVectorOCMComponent.Component, "VectorOCMComponent", vectorOCMComponentName)
	return baseVector.Artifacts, nil
}

func combineBaseArtifactsAndComponentArtifacts(baseArtifacts, componentArtifacts []domain.Artifact) []domain.Artifact {
	if len(baseArtifacts) == 0 {
		return componentArtifacts
	}

	for _, componentArtifact := range componentArtifacts {
		found := false
		for i, baseArtifact := range baseArtifacts {
			if componentArtifact.OcmReference.Component == baseArtifact.OcmReference.Component {
				baseArtifacts[i] = componentArtifact
				found = true
				break
			}
		}
		if !found {
			baseArtifacts = append(baseArtifacts, componentArtifact)
		}
	}
	return baseArtifacts
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
		For(&v1alpha1.VectorTemplate{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("vectortemplate").
		Complete(r)
}
