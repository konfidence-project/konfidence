package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/vector"
	"github.com/konfidence-project/konfidence/pkg/jsonschema"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	konfcompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	defaultReconcileInterval     = time.Minute
	VectorAssemblyControllerName = "vector-assembly-controller"
	EventActionStatusPatch       = "StatusPatch"
	EventActionDriftDetection    = "DriftDetection"
	EventActionVectorCreation    = "VectorCreation"
)

// errBaseVectorNotReady signals that the referenced base VectorTemplate has not assembled
// a vector yet. It is a normal transient state, not a failure: the caller stops the current
// reconcile without returning an error (no backoff) and relies on the base VectorTemplate
// watch to re-enqueue this template once the base's status.latestVector is populated.
var errBaseVectorNotReady = errors.New("base vector not ready")

// VectorTemplateReconciler reconciles a VectorTemplate object
type VectorTemplateReconciler struct {
	client.Client
	Recorder         events.EventRecorder
	Cache            *clientcache.Cache[*konfidence.VectorTemplate, vector.OcmPort]
	VersionGenerator vector.VersionGenerator
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectortemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectortemplates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile the VectorTemplate object to detect a vector drift and act upon it.
func (r *VectorTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	logf.IntoContext(ctx, log)
	log.Info("Reconciling VectorTemplate", "name", req.NamespacedName)

	vectorTemplate := &konfidence.VectorTemplate{}
	if err := r.Get(ctx, req.NamespacedName, vectorTemplate); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// preserve original vector template status for patching it later
	originalVectorTemplate := vectorTemplate.DeepCopy()
	patch := client.MergeFrom(originalVectorTemplate)

	driftErr := r.detectAndActOnDrift(ctx, vectorTemplate)
	// errBaseVectorNotReady is a normal transient state, not a failure. Swallow it so the
	// reconcile does not back off or record an error; the base VectorTemplate watch
	// re-enqueues this template once the base's status.latestVector is populated. The
	// informative waiting condition set on the status is still patched below.
	waitingForBase := errors.Is(driftErr, errBaseVectorNotReady)
	if waitingForBase {
		log.Info("waiting for base VectorTemplate to assemble a vector", "name", req.NamespacedName)
		driftErr = nil
	} else if driftErr != nil {
		log.Error(driftErr, "error detecting or acting on drift for Vector template")
	}

	// patch the vector template status updates (regardless of error in drift detection/handling)
	var patchErr error
	if !reflect.DeepEqual(vectorTemplate.Status, originalVectorTemplate.Status) {
		if patchErr = r.Status().Patch(ctx, vectorTemplate, patch); patchErr != nil {
			log.Error(patchErr, "unable to patch vector template status")
			r.Recorder.Eventf(vectorTemplate, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch, patchErr.Error())
		}
	}

	if err := errors.Join(driftErr, patchErr); err != nil {
		return ctrl.Result{}, err
	}
	// While waiting for the base to assemble, do not poll: the base VectorTemplate watch
	// re-enqueues this template the instant base.status.latestVector is set.
	if waitingForBase {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: requeueAfterFromSpecOrDefault(vectorTemplate)}, nil
}

func (r *VectorTemplateReconciler) detectAndActOnDrift(
	ctx context.Context,
	template *konfidence.VectorTemplate,
) error {
	log := logf.FromContext(ctx)

	ocmAdapter, err := r.Cache.Lookup(ctx, r.Client, template)
	if err != nil {
		err = fmt.Errorf("building OCM clients: %w", err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	ocmComponentsFromComponentList, err := mapComponentsToOCMReferences(template.Spec.Components)
	if err != nil {
		err = fmt.Errorf("unable to map vector template components to ocm references: %w", err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	vectorOCMComponent, err := konfcompref.Parse(
		template.Spec.UploadTarget, konfcompref.WithVersionValidation(konfcompref.VersionValidationNoVersion))
	if err != nil {
		err = fmt.Errorf("unable to create ocm reference from vector template upload target (%s): %w", template.Spec.UploadTarget, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	var desiredArtifacts []vector.Artifact
	if template.Spec.Base != nil {
		if desiredArtifacts, err = r.getArtifactsFromBaseVector(ctx, ocmAdapter, template, vectorOCMComponent.Component); err != nil {
			return err
		}
	}

	latestArtifactsFromComponentList, err := ocmAdapter.GetArtifacts(ctx, ocmComponentsFromComponentList)
	if err != nil {
		err = fmt.Errorf("unable to get desired artifacts for vector (%s): %w", vectorOCMComponent, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}

	desiredArtifacts = combineBaseArtifactsAndComponentArtifacts(desiredArtifacts, latestArtifactsFromComponentList)
	desiredVectorConfiguration, err := getVectorConfiguration(*template)
	if err != nil {
		err = fmt.Errorf("unable to build desired vector configuration: %w", err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return err
	}
	desiredVector := vector.Vector{
		Name:         vectorOCMComponent.Component,
		Artifacts:    desiredArtifacts,
		VectorConfig: desiredVectorConfiguration,
	}

	var currentVector vector.Vector
	// An empty latestVector means no vector has been assembled yet - treat as first build.
	if template.Status.LatestVector == "" {
		msg := "No latest vector recorded in status - creating new vector"
		r.Recorder.Eventf(template, nil, corev1.EventTypeNormal, "VectorNotFound", "ResolvingLatestVector", msg)
		log.Info(msg, "VectorOCMComponent", vectorOCMComponent.Component)
	} else {
		latestVectorRef, parseErr := konfcompref.Parse(
			template.Status.LatestVector, konfcompref.WithVersionValidation(konfcompref.VersionValidationSemverOnly))
		if parseErr != nil {
			err = fmt.Errorf("unable to parse status.latestVector (%s): %w", template.Status.LatestVector, parseErr)
			meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
				Type:               konfidence.VectorTemplateReadyCondition,
				Status:             metav1.ConditionUnknown,
				Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
				Message:            err.Error(),
				ObservedGeneration: template.Generation,
				LastTransitionTime: metav1.Now(),
			})
			r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
			return err
		}

		currentVector, err = ocmAdapter.GetVector(ctx, *latestVectorRef)
		if errors.Is(err, vector.ErrVectorNotFound) {
			msg := "Vector referenced by status.latestVector not found in OCM repository - creating new vector"
			r.Recorder.Eventf(template, nil, corev1.EventTypeNormal, "VectorNotFound", "ResolvingLatestVector", msg)
			log.Info(msg, "LatestVector", template.Status.LatestVector)
		} else if err != nil {
			err = fmt.Errorf("unable to get current artifacts from vector (%s): %w", latestVectorRef, err)
			meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
				Type:               konfidence.VectorTemplateReadyCondition,
				Status:             metav1.ConditionUnknown,
				Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
				Message:            err.Error(),
				ObservedGeneration: template.Generation,
				LastTransitionTime: metav1.Now(),
			})
			r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
			return err
		}
	}

	driftDetected := vector.HasDrift(currentVector, desiredVector)
	if !driftDetected {
		msg := fmt.Sprintf("No drift detected for vector - vector version is still %s", currentVector.Version)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             konfidence.VectorTemplateNoDriftDetectedReason,
			Message:            msg,
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeNormal, konfidence.VectorTemplateNoDriftDetectedReason, EventActionDriftDetection, msg)
		log.Info(msg, "VectorVersion", currentVector.Version, "VectorOCMComponent", vectorOCMComponent.Component)
		return nil
	}

	newVector := vector.Vector{
		Version:      r.VersionGenerator.Generate(),
		Name:         vectorOCMComponent.Component,
		Artifacts:    desiredArtifacts,
		VectorConfig: desiredVectorConfiguration,
	}

	err = ocmAdapter.CreateVector(ctx, vectorOCMComponent.Repository, newVector)
	if err != nil {
		err = fmt.Errorf("unable to create new vector (%s) on drift: %w", vectorOCMComponent, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             konfidence.VectorTemplateVectorCreationFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateVectorCreationFailedReason, EventActionVectorCreation, err.Error())
		return err
	}

	// Record the concrete version of the freshly assembled vector 
	latestVectorRef := compref.Ref{
		Repository: vectorOCMComponent.Repository,
		Component:  vectorOCMComponent.Component,
		Version:    newVector.Version,
	}
	template.Status.LatestVector = latestVectorRef.String()

	msg := fmt.Sprintf("Drift detected and new vector created successfully - new vector version is %s", newVector.Version)
	meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorTemplateReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.VectorTemplateVectorCreatedReason,
		Message:            msg,
		ObservedGeneration: template.Generation,
		LastTransitionTime: metav1.Now(),
	})
	r.Recorder.Eventf(template, nil, corev1.EventTypeNormal, konfidence.VectorTemplateVectorCreatedReason, "VectorCreation", msg)
	log.Info(msg, "VectorVersion", newVector.Version, "VectorOCMComponent", vectorOCMComponent.Component)
	return nil
}

func (r *VectorTemplateReconciler) getArtifactsFromBaseVector(
	ctx context.Context, ocmAdapter vector.OcmPort, template *konfidence.VectorTemplate,
	vectorOCMComponentName string,
) ([]vector.Artifact, error) {
	// Resolve the base by reading the referenced VectorTemplate's status.latestVector,
	// which holds the concrete version of its most recently assembled vector.
	baseTemplate := &konfidence.VectorTemplate{}
	baseKey := types.NamespacedName{Namespace: template.Namespace, Name: template.Spec.Base.Name}
	if err := r.Get(ctx, baseKey, baseTemplate); err != nil {
		err = fmt.Errorf("unable to get base VectorTemplate (%s): %w", template.Spec.Base.Name, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return nil, err
	}

	// The base has not assembled a vector yet. This is a normal transient state, not an
	// error: set a waiting condition and return the sentinel so the caller stops without
	// backoff. The Watches mapping on base VectorTemplates re-enqueues this template as
	// soon as the base's status.latestVector is populated.
	if baseTemplate.Status.LatestVector == "" {
		msg := fmt.Sprintf("waiting for base VectorTemplate (%s) to assemble a vector", template.Spec.Base.Name)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             konfidence.VectorTemplateWaitingForBaseReason,
			Message:            msg,
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeNormal, konfidence.VectorTemplateWaitingForBaseReason, EventActionDriftDetection, msg)
		return nil, errBaseVectorNotReady
	}

	baseVectorOCMComponent, err := konfcompref.Parse(
		baseTemplate.Status.LatestVector, konfcompref.WithVersionValidation(konfcompref.VersionValidationSemverOnly))
	if err != nil {
		err = fmt.Errorf("unable to create ocm reference from base VectorTemplate (%s) status.latestVector (%s): %w",
			template.Spec.Base.Name, baseTemplate.Status.LatestVector, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return nil, err
	}

	baseVector, err := ocmAdapter.GetVector(ctx, *baseVectorOCMComponent)
	if err != nil {
		err = fmt.Errorf("unable to get artifacts from base vector (%s): %w", baseVectorOCMComponent, err)
		meta.SetStatusCondition(&template.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionUnknown,
			Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
			Message:            err.Error(),
			ObservedGeneration: template.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(template, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
		return nil, err
	}
	log := logf.FromContext(ctx)
	log.Info("Using base vector for vector OCM component",
		"BaseVectorVersion", baseVector.Version,
		"BaseVectorOCMComponent", baseVectorOCMComponent.Component,
		"VectorOCMComponent", vectorOCMComponentName)

	return baseVector.Artifacts, nil
}

func combineBaseArtifactsAndComponentArtifacts(baseArtifacts, componentArtifacts []vector.Artifact) []vector.Artifact {
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

func mapComponentsToOCMReferences(components []konfidence.Component) ([]compref.Ref, error) {
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

func requeueAfterFromSpecOrDefault(vectorTemplate *konfidence.VectorTemplate) time.Duration {
	if vectorTemplate.Spec.ReconcileInterval != nil {
		return vectorTemplate.Spec.ReconcileInterval.Duration
	}
	return defaultReconcileInterval
}

func getVectorConfiguration(vectorTemplate konfidence.VectorTemplate) (*vector.VectorConfiguration, error) {
	if vectorTemplate.Spec.VectorConfig == nil ||
		(vectorTemplate.Spec.VectorConfig.Features == nil && vectorTemplate.Spec.VectorConfig.Authored == nil) {
		return nil, nil
	}

	vectorConfig := vectorTemplate.Spec.VectorConfig
	var features json.RawMessage
	if vectorConfig.Features != nil {
		features = vectorConfig.Features.Raw
	}
	var authored json.RawMessage
	if vectorConfig.Authored != nil {
		authored = vectorConfig.Authored.Raw
	}
	vectorConfigSchema := jsonschema.NewVectorConfigurationV1(features, authored)

	content, err := json.Marshal(vectorConfigSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize vectorConfigSchema: %w", err)
	}

	return &vector.VectorConfiguration{
		Content: content,
	}, nil
}

// NewVectorTemplateReconciler wires a VectorTemplateReconciler for the given manager.
func NewVectorTemplateReconciler(
	mgr ctrl.Manager,
	cache *clientcache.Cache[*konfidence.VectorTemplate, vector.OcmPort],
	versionGenerator vector.VersionGenerator,
) *VectorTemplateReconciler {
	return &VectorTemplateReconciler{
		Client:           mgr.GetClient(),
		Recorder:         mgr.GetEventRecorder(VectorAssemblyControllerName),
		Cache:            cache,
		VersionGenerator: versionGenerator,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorTemplate{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&konfidence.VectorTemplate{},
			handler.EnqueueRequestsFromMapFunc(r.mapBaseToDependents),
			builder.WithPredicates(latestVectorChangedPredicate()),
		).
		Named("vectortemplate").
		Complete(r)
}

// latestVectorChangedPredicate fires the base-VectorTemplate watch only when a base's
// status.latestVector actually changes on update - the sole event that gives a waiting
// dependent something new to assemble against. Creates carry no signal (status.latestVector
// is empty at creation and only ever populated by a later status update; a dependent's own
// creation is handled by the For source), so they, deletes, and generic/no-op status churn
// are all dropped. This bounds both the mapper's namespace List and dependent re-enqueues.
func latestVectorChangedPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(event.CreateEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldTemplate, okOld := e.ObjectOld.(*konfidence.VectorTemplate)
			newTemplate, okNew := e.ObjectNew.(*konfidence.VectorTemplate)
			if !okOld || !okNew {
				return false
			}
			return oldTemplate.Status.LatestVector != newTemplate.Status.LatestVector
		},
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

// mapBaseToDependents enqueues every VectorTemplate in the same namespace that references
// the changed VectorTemplate as its base. This wakes dependents as soon as a base assembles
// or reassembles a vector (its status.latestVector changes) instead of relying on the periodic requeue interval.
func (r *VectorTemplateReconciler) mapBaseToDependents(ctx context.Context, obj client.Object) []ctrl.Request {
	base, ok := obj.(*konfidence.VectorTemplate)
	if !ok {
		return nil
	}

	dependents := &konfidence.VectorTemplateList{}
	if err := r.List(ctx, dependents, client.InNamespace(base.Namespace)); err != nil {
		logf.FromContext(ctx).Error(err, "unable to list VectorTemplates to map base to dependents",
			"base", base.Name, "namespace", base.Namespace)
		return nil
	}

	var requests []ctrl.Request
	for i := range dependents.Items {
		dependent := &dependents.Items[i]
		if dependent.Spec.Base != nil && dependent.Spec.Base.Name == base.Name {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{Namespace: dependent.Namespace, Name: dependent.Name},
			})
		}
	}
	return requests
}
