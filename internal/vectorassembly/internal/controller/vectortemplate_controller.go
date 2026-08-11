package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	lru "github.com/hashicorp/golang-lru/v2"
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
	defaultAssemblyPollInterval  = 5 * time.Second
	VectorAssemblyControllerName = "vector-assembly-controller"
	EventActionStatusPatch       = "StatusPatch"
	EventActionDriftDetection    = "DriftDetection"
	EventActionVectorCreation    = "VectorCreation"
	VectorCacheSize              = 2048
)

var (
	// errBaseVectorNotReady signals that the referenced base VectorTemplate has not assembled
	// a vector yet. It is a normal transient state, not a failure: the caller stops the current
	// reconcile without returning an error (no backoff) and relies on the base VectorTemplate
	// watch to re-enqueue this template once the base's status.latestVector is populated.
	errBaseVectorNotReady = errors.New("base vector not ready")

	errDriftDetectionFailed = errors.New("drift detection failed")
)

// VectorTemplateReconciler reconciles a VectorTemplate object
type VectorTemplateReconciler struct {
	client.Client
	Recorder             events.EventRecorder
	Cache                *clientcache.Cache[*konfidence.VectorTemplate, vector.OcmPort]
	VectorCache          *lru.Cache[string, vector.Vector]
	VersionGenerator     vector.VersionGenerator
	jobs                 *jobRegistry
	assemblyPollInterval time.Duration
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

	result, reconcileErr := r.reconcileAsync(ctx, req.NamespacedName, vectorTemplate)

	// errBaseVectorNotReady is a normal transient state, not a failure. Swallow it so the
	// reconcile does not back off or record an error; the base VectorTemplate watch
	// re-enqueues this template once the base's status.latestVector is populated. The
	// informative waiting condition set on the status is still patched below.
	waitingForBase := errors.Is(reconcileErr, errBaseVectorNotReady)
	if waitingForBase {
		log.Info("waiting for base VectorTemplate to assemble a vector", "name", req.NamespacedName)
		reconcileErr = nil
	} else if reconcileErr != nil {
		log.Error(reconcileErr, "error detecting or acting on drift for Vector template")
	}

	// patch the vector template status updates (regardless of error in drift detection/handling)
	var patchErr error
	if !reflect.DeepEqual(vectorTemplate.Status, originalVectorTemplate.Status) {
		if patchErr = r.Status().Patch(ctx, vectorTemplate, patch); patchErr != nil {
			log.Error(patchErr, "unable to patch vector template status")
			r.Recorder.Eventf(vectorTemplate, nil, corev1.EventTypeWarning, "StatusPatchFailed", EventActionStatusPatch, patchErr.Error())
		}
	}

	if err := errors.Join(reconcileErr, patchErr); err != nil {
		return ctrl.Result{}, err
	}
	// While waiting for the base to assemble, do not poll: the base VectorTemplate watch
	// re-enqueues this template the instant base.status.latestVector is set.
	if waitingForBase {
		return ctrl.Result{}, nil
	}
	return result, nil
}

func (r *VectorTemplateReconciler) reconcileAsync(
	ctx context.Context,
	nn types.NamespacedName,
	vt *konfidence.VectorTemplate,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	ocmAdapter, err := r.Cache.Lookup(ctx, r.Client, vt)
	if err != nil {
		return ctrl.Result{}, r.setDriftDetectionFailed(vt, fmt.Errorf("building OCM clients: %w", err))
	}

	ocmComponentRefs, err := mapComponentsToOCMReferences(vt.Spec.Components)
	if err != nil {
		return ctrl.Result{}, r.setDriftDetectionFailed(vt,
			fmt.Errorf("unable to map vector template components to ocm references: %w", err))
	}

	vectorOCMComponent, err := konfcompref.Parse(
		vt.Spec.UploadTarget, konfcompref.WithVersionValidation(konfcompref.VersionValidationNoVersion))
	if err != nil {
		return ctrl.Result{}, r.setDriftDetectionFailed(vt,
			fmt.Errorf("unable to create ocm reference from vector template upload target (%s): %w", vt.Spec.UploadTarget, err))
	}

	var baseVectorRef *compref.Ref
	if vt.Spec.Base != nil {
		ref, baseErr := r.resolveBaseRef(ctx, vt)
		if baseErr != nil {
			return ctrl.Result{}, baseErr
		}
		baseVectorRef = ref
	}

	// Parse the current vector ref from status.latestVector.
	var currentVectorRef *compref.Ref
	if vt.Status.LatestVector != "" {
		ref, parseErr := konfcompref.Parse(
			vt.Status.LatestVector, konfcompref.WithVersionValidation(konfcompref.VersionValidationSemverOnly))
		if parseErr != nil {
			return ctrl.Result{}, r.setDriftDetectionFailed(vt,
				fmt.Errorf("unable to parse status.latestVector (%s): %w", vt.Status.LatestVector, parseErr))
		}
		currentVectorRef = ref
	}

	desiredVectorConfig, err := getVectorConfiguration(*vt)
	if err != nil {
		return ctrl.Result{}, r.setDriftDetectionFailed(vt,
			fmt.Errorf("unable to build desired vector configuration: %w", err))
	}

	// Check for an inflight job.
	job, exists := r.jobs.get(nn)
	if exists {
		// Stale generation — cancel it and fall through to launch a new one.
		if job.generation != vt.Generation {
			log.Info("cancelling stale inflight assembly job",
				"jobGeneration", job.generation, "currentGeneration", vt.Generation)
			r.jobs.remove(nn)
		} else if !job.done() {
			// Still running for the current generation — poll later.
			return ctrl.Result{RequeueAfter: r.assemblyPollInterval}, nil
		} else { //nolint:gocritic // cleaner to keep as if-else chain
			// Finished — apply the result.
			res := <-job.result
			r.jobs.remove(nn)
			return r.applyAssemblyResult(vt, res, log)
		}
	}

	// Launch a new assembly job.
	r.jobs.launch(nn, vt.Generation, func(jobCtx context.Context) assemblyResult {
		return r.runAssembly(jobCtx, ocmAdapter, ocmComponentRefs,
			vectorOCMComponent, baseVectorRef, currentVectorRef, desiredVectorConfig)
	})
	log.Info("launched background assembly job", "generation", vt.Generation)
	return ctrl.Result{RequeueAfter: r.assemblyPollInterval}, nil
}

func (r *VectorTemplateReconciler) runAssembly(
	ctx context.Context,
	adapter vector.OcmPort,
	componentRefs []compref.Ref,
	uploadTarget *compref.Ref,
	baseVectorRef *compref.Ref,
	currentVectorRef *compref.Ref,
	desiredVectorConfig *vector.VectorConfiguration,
) assemblyResult {
	// Resolve base vector artifacts (if applicable).
	var baseArtifacts []vector.Artifact
	if baseVectorRef != nil {
		baseVector, err := r.getVectorCached(ctx, adapter, *baseVectorRef)
		if err != nil {
			return assemblyResult{err: fmt.Errorf("unable to get artifacts from base vector (%s): %w %w", baseVectorRef, err, errDriftDetectionFailed)}
		}
		baseArtifacts = baseVector.Artifacts
	}

	// Resolve current vector (for drift comparison).
	var currentVector vector.Vector
	if currentVectorRef != nil {
		cv, err := r.getVectorCached(ctx, adapter, *currentVectorRef)
		if err != nil && !errors.Is(err, vector.ErrVectorNotFound) {
			return assemblyResult{err: fmt.Errorf("unable to get current artifacts from vector (%s): %w %w", currentVectorRef, err, errDriftDetectionFailed)}
		}
		currentVector = cv
	}

	// Fetch upstream component artifacts (always OCI — this detects upstream changes).
	componentArtifacts, err := adapter.GetArtifacts(ctx, componentRefs)
	if err != nil {
		return assemblyResult{err: fmt.Errorf("unable to get desired artifacts for vector (%s): %w %w", uploadTarget, err, errDriftDetectionFailed)}
	}

	// Combine base + component artifacts and check for drift.
	desiredArtifacts := combineBaseArtifactsAndComponentArtifacts(baseArtifacts, componentArtifacts)
	desiredVector := vector.Vector{
		Name:         uploadTarget.Component,
		Artifacts:    desiredArtifacts,
		VectorConfig: desiredVectorConfig,
	}

	if !vector.HasDrift(currentVector, desiredVector) {
		return assemblyResult{vectorVersion: currentVector.Version, componentName: uploadTarget.Component}
	}

	// Drift detected — create a new vector.
	newVersion := r.VersionGenerator.Generate()
	newVector := vector.Vector{
		Version:      newVersion,
		Name:         uploadTarget.Component,
		Artifacts:    desiredArtifacts,
		VectorConfig: desiredVectorConfig,
	}

	if err = adapter.CreateVector(ctx, uploadTarget.Repository, newVector); err != nil {
		return assemblyResult{err: fmt.Errorf("unable to create new vector (%s) on drift: %w", uploadTarget, err)}
	}

	latestRef := compref.Ref{
		Repository: uploadTarget.Repository,
		Component:  uploadTarget.Component,
		Version:    newVersion,
	}
	return assemblyResult{
		latestVector:  latestRef.String(),
		vectorVersion: newVersion,
		componentName: uploadTarget.Component,
	}
}

// applyAssemblyResult maps a completed assemblyResult onto the VectorTemplate status
// conditions and events.
func (r *VectorTemplateReconciler) applyAssemblyResult(
	vt *konfidence.VectorTemplate,
	res assemblyResult,
	log logr.Logger,
) (ctrl.Result, error) {
	switch {
	case res.failed():
		reason := konfidence.VectorTemplateVectorCreationFailedReason
		status := metav1.ConditionFalse
		eventAction := EventActionVectorCreation
		if errors.Is(res.err, errDriftDetectionFailed) {
			reason = konfidence.VectorTemplateDriftDetectionFailedReason
			status = metav1.ConditionUnknown
			eventAction = EventActionDriftDetection
		}
		meta.SetStatusCondition(&vt.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             status,
			Reason:             reason,
			Message:            res.err.Error(),
			ObservedGeneration: vt.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(vt, nil, corev1.EventTypeWarning, reason, eventAction, res.err.Error())
		log.Error(res.err, "assembly failed", "component", res.componentName)
		return ctrl.Result{}, res.err

	case res.created():
		vt.Status.LatestVector = res.latestVector
		msg := fmt.Sprintf("Drift detected and new vector created successfully - new vector version is %s", res.vectorVersion)
		meta.SetStatusCondition(&vt.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             konfidence.VectorTemplateVectorCreatedReason,
			Message:            msg,
			ObservedGeneration: vt.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(vt, nil, corev1.EventTypeNormal, konfidence.VectorTemplateVectorCreatedReason, EventActionVectorCreation, msg)
		log.Info(msg, "VectorVersion", res.vectorVersion, "VectorOCMComponent", res.componentName)

	default: // noDrift
		msg := fmt.Sprintf("No drift detected for vector - vector version is still %s", res.vectorVersion)
		meta.SetStatusCondition(&vt.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             konfidence.VectorTemplateNoDriftDetectedReason,
			Message:            msg,
			ObservedGeneration: vt.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(vt, nil, corev1.EventTypeNormal, konfidence.VectorTemplateNoDriftDetectedReason, EventActionDriftDetection, msg)
		log.Info(msg, "VectorVersion", res.vectorVersion, "VectorOCMComponent", res.componentName)
	}

	return ctrl.Result{RequeueAfter: requeueAfterFromSpecOrDefault(vt)}, nil
}

func (r *VectorTemplateReconciler) setDriftDetectionFailed(vt *konfidence.VectorTemplate, err error) error {
	meta.SetStatusCondition(&vt.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorTemplateReadyCondition,
		Status:             metav1.ConditionUnknown,
		Reason:             konfidence.VectorTemplateDriftDetectionFailedReason,
		Message:            err.Error(),
		ObservedGeneration: vt.Generation,
		LastTransitionTime: metav1.Now(),
	})
	r.Recorder.Eventf(vt, nil, corev1.EventTypeWarning, konfidence.VectorTemplateDriftDetectionFailedReason, EventActionDriftDetection, err.Error())
	return err
}

// resolveBaseRef reads the base VectorTemplate's status.latestVector and returns it
// as a parsed compref.Ref. Returns errBaseVectorNotReady if the base hasn't assembled yet.
func (r *VectorTemplateReconciler) resolveBaseRef(ctx context.Context, vt *konfidence.VectorTemplate) (*compref.Ref, error) {
	baseTemplate := &konfidence.VectorTemplate{}
	baseKey := types.NamespacedName{Namespace: vt.Namespace, Name: vt.Spec.Base.Name}
	if err := r.Get(ctx, baseKey, baseTemplate); err != nil {
		return nil, r.setDriftDetectionFailed(vt,
			fmt.Errorf("unable to get base VectorTemplate (%s): %w", vt.Spec.Base.Name, err))
	}

	// The base has not assembled a vector yet. This is a normal transient state, not an
	// error: set a waiting condition and return the sentinel so the caller stops without
	// backoff. The Watches mapping on base VectorTemplates re-enqueues this template as
	// soon as the base's status.latestVector is populated.
	if baseTemplate.Status.LatestVector == "" {
		msg := fmt.Sprintf("waiting for base VectorTemplate (%s) to assemble a vector", vt.Spec.Base.Name)
		meta.SetStatusCondition(&vt.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorTemplateReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             konfidence.VectorTemplateWaitingForBaseReason,
			Message:            msg,
			ObservedGeneration: vt.Generation,
			LastTransitionTime: metav1.Now(),
		})
		r.Recorder.Eventf(vt, nil, corev1.EventTypeNormal, konfidence.VectorTemplateWaitingForBaseReason, EventActionDriftDetection, msg)
		return nil, errBaseVectorNotReady
	}

	ref, err := konfcompref.Parse(
		baseTemplate.Status.LatestVector, konfcompref.WithVersionValidation(konfcompref.VersionValidationSemverOnly))
	if err != nil {
		return nil, r.setDriftDetectionFailed(vt,
			fmt.Errorf("unable to create ocm reference from base VectorTemplate (%s) status.latestVector (%s): %w",
				vt.Spec.Base.Name, baseTemplate.Status.LatestVector, err))
	}
	return ref, nil
}

// getVectorCached returns a verified vector from the LRU, calling
// adapter.GetVector (OCI fetch + signature verify) only on a cache miss.
// ErrVectorNotFound propagates uncached so the caller triggers a new assembly.
func (r *VectorTemplateReconciler) getVectorCached(ctx context.Context, adapter vector.OcmPort, ref compref.Ref) (vector.Vector, error) {
	key := ref.String()
	if v, ok := r.VectorCache.Get(key); ok {
		return v, nil
	}
	v, err := adapter.GetVector(ctx, ref)
	if err != nil {
		return vector.Vector{}, err
	}
	r.VectorCache.Add(key, v)
	return v, nil
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
	vectorCache *lru.Cache[string, vector.Vector],
	versionGenerator vector.VersionGenerator,
) *VectorTemplateReconciler {
	return &VectorTemplateReconciler{
		Client:               mgr.GetClient(),
		Recorder:             mgr.GetEventRecorder(VectorAssemblyControllerName),
		Cache:                cache,
		VectorCache:          vectorCache,
		VersionGenerator:     versionGenerator,
		jobs:                 newJobRegistry(),
		assemblyPollInterval: defaultAssemblyPollInterval,
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
