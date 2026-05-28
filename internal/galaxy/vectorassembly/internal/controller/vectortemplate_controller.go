package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	domain2 "github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/vector"
	konfcompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"
)

const (
	defaultReconcileInterval = time.Minute

	VectorAssemblyControllerName = "galaxy-vector-assembly-controller"

	EventActionStatusPatch    = "StatusPatch"
	EventActionDriftDetection = "DriftDetection"
	EventActionVectorCreation = "VectorCreation"
)

// VectorTemplateReconciler reconciles a VectorTemplate object
type VectorTemplateReconciler struct {
	Mgr                   mcmanager.Manager
	Scheme                *runtime.Scheme
	OcmClientProvider     repository.ClientProvider
	VectorOcmPortProvider domain2.OcmPortProvider
	VersionGenerator      domain2.VersionGenerator
}

// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectortemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=vectortemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile the VectorTemplate object to detect a vector drift and act upon it.
func (r *VectorTemplateReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx).WithValues("cluster", req.ClusterName)
	logf.IntoContext(ctx, log)
	log.Info("Reconciling VectorTemplate", "name", req.NamespacedName)

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	clusterClient := cluster.GetClient()
	recorder := cluster.GetEventRecorder(VectorAssemblyControllerName)

	vectorTemplate := &v1alpha1.VectorTemplate{}
	if err := clusterClient.Get(ctx, req.NamespacedName, vectorTemplate); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// preserve original vector template status for patching it later
	originalVectorTemplate := vectorTemplate.DeepCopy()
	patch := client.MergeFrom(originalVectorTemplate)

	if err = r.detectAndActOnDrift(ctx, clusterClient, vectorTemplate, recorder); err != nil {
		log.Error(err, "error detecting or acting on drift for Vector template")
	}

	// patch the vector template status updates (regardless of error in drift detection/handling)
	var patchErr error
	if !reflect.DeepEqual(vectorTemplate.Status, originalVectorTemplate.Status) {
		if patchErr = clusterClient.Status().Patch(ctx, vectorTemplate, patch); patchErr != nil {
			log.Error(patchErr, "unable to patch vector template status")
			recorder.Eventf(vectorTemplate, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch, patchErr.Error())
		}
	}

	if err = errors.Join(err, patchErr); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: requeueAfterFromSpecOrDefault(vectorTemplate)}, nil
}

func (r *VectorTemplateReconciler) detectAndActOnDrift(
	ctx context.Context, clusterClient client.Client,
	template *v1alpha1.VectorTemplate, recorder events.EventRecorder,
) error {
	log := logf.FromContext(ctx)

	ocmClient, err := r.OcmClientProvider.NewClient(ctx, clusterClient, template.GetNamespace(), template.Spec.Config)
	if err != nil {
		return fmt.Errorf("unable to create OCM client: %w", err)
	}

	ocmAdapter := r.VectorOcmPortProvider.NewVectorOcmPort(ocmClient)

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
		recorder.Eventf(template, nil, corev1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	vectorOCMComponent, err := konfcompref.Parse(
		template.Spec.UploadTarget, konfcompref.WithVersionValidation(konfcompref.VersionValidationAliasOnly))
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
		recorder.Eventf(template, nil, corev1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	var desiredArtifacts []domain2.Artifact
	if template.Spec.Base != nil {
		if desiredArtifacts, err = r.getArtifactsFromBaseVector(ctx, ocmAdapter, template, vectorOCMComponent.Component, recorder); err != nil {
			return err
		}
	}

	latestArtifactsFromComponentList, err := ocmAdapter.GetArtifacts(ctx, ocmComponentsFromComponentList)
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
		recorder.Eventf(template, nil, corev1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	desiredArtifacts = combineBaseArtifactsAndComponentArtifacts(desiredArtifacts, latestArtifactsFromComponentList)

	actualVector, err := ocmAdapter.GetVector(ctx, *vectorOCMComponent)
	if errors.Is(err, domain2.ErrVectorNotFound) {
		msg := "Vector not found in OCM repository - creating new vector"
		recorder.Eventf(template, nil, corev1.EventTypeNormal, "VectorNotFound", "ResolvingLatestVector", msg)
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
		recorder.Eventf(template, nil, corev1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	driftDetected := domain2.HasDrift(desiredArtifacts, actualVector.Artifacts)
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
		recorder.Eventf(template, nil, corev1.EventTypeNormal, v1alpha1.VectorTemplateNoDriftDetectedReason, EventActionDriftDetection, msg)
		log.Info(msg, "VectorVersion", actualVector.Version, "VectorOCMComponent", vectorOCMComponent.Component)
		return nil
	}

	newVector := domain2.Vector{
		Version:   r.VersionGenerator.Generate(),
		Name:      vectorOCMComponent.Component,
		Artifacts: desiredArtifacts,
	}

	err = ocmAdapter.CreateVector(ctx, vectorOCMComponent.Repository, newVector, vectorOCMComponent.Version)
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
		recorder.Eventf(template, nil, corev1.EventTypeWarning, v1alpha1.VectorTemplateVectorCreationFailedReason, EventActionVectorCreation, err.Error())
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
	recorder.Eventf(template, nil, corev1.EventTypeNormal, v1alpha1.VectorTemplateVectorCreatedReason, "VectorCreation", msg)
	log.Info(msg, "VectorVersion", newVector.Version, "VectorOCMComponent", vectorOCMComponent.Component)
	return nil
}

func (r *VectorTemplateReconciler) getArtifactsFromBaseVector(
	ctx context.Context, ocmAdapter domain2.OcmPort, template *v1alpha1.VectorTemplate,
	vectorOCMComponentName string, recorder events.EventRecorder,
) ([]domain2.Artifact, error) {
	baseVectorOCMComponent, err := konfcompref.Parse(*template.Spec.Base)
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
		recorder.Eventf(template, nil, corev1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return nil, err
	}

	baseVector, err := ocmAdapter.GetVector(ctx, *baseVectorOCMComponent)
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
		recorder.Eventf(template, nil, corev1.EventTypeWarning, v1alpha1.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return nil, err
	}
	log := logf.FromContext(ctx)
	log.Info("Using base vector for vector OCM component",
		"BaseVectorVersion", baseVector.Version,
		"BaseVectorOCMComponent", baseVectorOCMComponent.Component,
		"VectorOCMComponent", vectorOCMComponentName)

	return baseVector.Artifacts, nil
}

func combineBaseArtifactsAndComponentArtifacts(baseArtifacts, componentArtifacts []domain2.Artifact) []domain2.Artifact {
	if len(baseArtifacts) == 0 {
		return componentArtifacts
	}

	for _, componentArtifact := range componentArtifacts {
		found := false
		for i, baseArtifact := range baseArtifacts {
			if componentArtifact.Name == baseArtifact.Name {
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

func mapComponentsToOCMReferences(components []v1alpha1.Component) ([]compref.Ref, error) {
	// add component names to map to remove duplicates
	seen := make(map[string]struct{}, len(components))
	componentNames := make([]string, 0, len(components))

	for _, component := range components {
		if _, ok := seen[component.Name]; ok {
			continue
		}
		seen[component.Name] = struct{}{}
		componentNames = append(componentNames, component.Name)
	}

	ocmRefs := make([]compref.Ref, 0, len(componentNames))
	for _, componentName := range componentNames {
		componentOcmRef, err := konfcompref.Parse(componentName)
		if err != nil {
			return nil, fmt.Errorf("unable to create ocm reference from vector template component (%s): %w",
				componentName, err)
		}
		ocmRefs = append(ocmRefs, *componentOcmRef)
	}
	return ocmRefs, nil
}

func requeueAfterFromSpecOrDefault(vectorTemplate *v1alpha1.VectorTemplate) time.Duration {
	if vectorTemplate.Spec.ReconcileInterval != nil {
		return vectorTemplate.Spec.ReconcileInterval.Duration
	}
	return defaultReconcileInterval
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorTemplateReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		For(&v1alpha1.VectorTemplate{}, mcbuilder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("vectortemplate").
		Complete(r)
}
