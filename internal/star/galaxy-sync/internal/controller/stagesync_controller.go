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
	"strings"
	"time"

	"github.com/go-logr/logr"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	syncControllerFinalizer  = "konfidence.cloud/stage-sync-finalizer"
	defaultReconcileInterval = 30 * time.Second
	deletionRequeueInterval  = 5 * time.Second
	managedByLabelKey        = "app.kubernetes.io/managed-by"
	galaxyStageSyncLabelKey   = "konfidence.cloud/galaxy-stage-sync"
	stageSyncedByLabelPrefix = "synced-by-star/"
	StageSyncControllerName  = "stage-sync-controller"
)

// StageSyncReconciler watches a StageSync object on the remote client (galaxy) and creates/updates/deletes a corresponding Stage object on the local cluster (star).
type StageSyncReconciler struct {
	// LocalClient is the client accessing the star
	LocalClient client.Client
	// RemoteCluster is the cluster accessor for the galaxy side. Its cache is
	// managed by the controller-runtime manager, which guarantees it is
	// started (and synced) before any informers sourced from it.
	RemoteCluster cluster.Cluster
	Scheme        *runtime.Scheme
	Recorder      events.EventRecorder
	LandscapeName string // name of the local (star) cluster (landscape name), used for labeling
}

// +kubebuilder:rbac:groups="",resources=namespaces;secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stagesyncs;stagesyncs/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

func (r *StageSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling StageSync")

	// fetch remote Resource
	remoteStageSync := &global.StageSync{}
	err := r.RemoteCluster.GetClient().Get(ctx, req.NamespacedName, remoteStageSync)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch remote resource")
		return ctrl.Result{}, err
	}
	originalRemoteStageSync := remoteStageSync.DeepCopy()
	stageTemplate, err := getStageFromTemplate(remoteStageSync.Spec.StageTemplate)
	if err != nil {
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.InvalidStageTemplateReason, err.Error())
		err = errors.Join(err, patchErr)
		return ctrl.Result{}, err
	}
	reconcileInterval := remoteStageSync.Spec.ReconcileInterval
	if reconcileInterval == nil {
		reconcileInterval = &metav1.Duration{Duration: defaultReconcileInterval}
	}

	// check if the stored and served storage versions contain the one in the StageTemplate
	versionServed, err := r.isStageVersionServed(ctx, logger, stageTemplate.TypeMeta.GroupVersionKind().Version)
	if err != nil {
		msg := fmt.Sprintf("unable to verify if stage version is served: %s", err)
		logger.Error(err, msg)
		r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeWarning, global.StageCrdQueryFailedReason, "VerifyStageVersion", msg)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.StageCrdQueryFailedReason, msg)
		return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
	}
	if !versionServed {
		msg := fmt.Sprintf("expected version %s is not present or not served in Stage CRD", stageTemplate.TypeMeta.GroupVersionKind().Version)
		logger.Error(nil, msg)
		r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeWarning, global.APIVersionNotSupportedReason, "VerifyStageVersion", msg)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.APIVersionNotSupportedReason, msg)
		return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
	}

	// check if namespace exists on local
	if err := r.LocalClient.Get(ctx, client.ObjectKey{Name: stageTemplate.Namespace}, &corev1.Namespace{}); err != nil {
		msg := fmt.Sprintf("unable to fetch local namespace %s: %s", stageTemplate.Namespace, err)
		logger.Error(err, msg)
		r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeWarning, global.NamespaceNotFoundReason, "CheckLocalNamespace", msg)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.NamespaceNotFoundReason, msg)
		return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
	}

	// fetch local resource
	localStage := &landscape.Stage{}
	err = r.LocalClient.Get(ctx, client.ObjectKey{Name: stageTemplate.Name, Namespace: stageTemplate.Namespace}, localStage)
	if err != nil && !apierrors.IsNotFound(err) {
		err = fmt.Errorf("unable to fetch local resource: %w", err)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.StageQueryFailedReason, err.Error())
		err = errors.Join(err, patchErr)
		return ctrl.Result{}, err
	}
	// err is either nil (stage exists) or NotFound
	if !apierrors.IsNotFound(err) {
		// stage found: check if existing stage is managed by this StageSync resource
		managedBy, hasManagedBy := localStage.GetLabels()[managedByLabelKey]
		parentStageSync, hasParentStageSync := localStage.GetLabels()[galaxyStageSyncLabelKey]
		if !hasManagedBy || managedBy != StageSyncControllerName ||
			!hasParentStageSync || parentStageSync != sanitizeLabelValue(req.String()) {
			msg := fmt.Sprintf("stage that is not managed by this StageSync resource already exists with desired name and namespace; expected labels '%s: %s' and '%s: %s'",
				managedByLabelKey, StageSyncControllerName,
				galaxyStageSyncLabelKey, sanitizeLabelValue(req.String()))
			logger.Error(nil, msg)
			r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeWarning, global.ConflictWithUnmanagedStageReason, "CheckManagedStage", msg)
			patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.ConflictWithUnmanagedStageReason, msg)
			return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
		}
	}

	// handle deletion
	if !remoteStageSync.DeletionTimestamp.IsZero() {
		result, err := r.handleStageSyncDeletion(ctx, remoteStageSync, originalRemoteStageSync, localStage)
		return result, err
	}
	if err := r.ensureFinalizer(ctx, remoteStageSync, originalRemoteStageSync); err != nil {
		return ctrl.Result{}, err
	}

	// label the remote StageSync with the local cluster name so it is visible
	// on the galaxy side which star cluster is syncing it
	if err := r.ensureStageSyncedByLabel(ctx, remoteStageSync, originalRemoteStageSync); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to set stage-synced-by label on remote StageSync: %w", err)
	}

	// adjust local stage based on remote stage template
	adjustStageFromTemplate(localStage, stageTemplate, req.NamespacedName)

	// create or update local resource
	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.LocalClient, localStage, func() error {
		localStage.Spec = stageTemplate.Spec
		return nil
	})
	if err != nil {
		msg := fmt.Sprintf("unable to create or update local resource: %s", err)
		logger.Error(err, msg)
		r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeWarning, global.StageReconcileFailedReason, "CreateOrUpdateStage", msg)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.StageReconcileFailedReason, msg)
		return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
	}

	// reflect status of local stage on remote resource
	if err := reflectStageStatusOnStageSync(localStage, remoteStageSync); err != nil {
		msg := fmt.Sprintf("unable to reflect stage status on StageSync: %s", err)
		logger.Error(err, msg)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.StageStatusReflectionFailedReason, msg)
		return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
	}

	// update status of remote resource
	msg := fmt.Sprintf("stage %s/%s reconciled successfully (operation: %s)", localStage.Namespace, localStage.Name, operationResult)
	patchErr := r.setAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, metav1.ConditionTrue, global.StageReconcileSuccessfulReason, msg)
	if patchErr != nil {
		err = fmt.Errorf("unable to patch status of remote resource: %w", patchErr)
		return ctrl.Result{}, err
	}

	logger.Info(msg)
	r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeNormal, global.StageReconcileSuccessfulReason, "CreateOrUpdateStage", msg)

	return ctrl.Result{RequeueAfter: reconcileInterval.Duration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapStageSyncToRequests := func(ctx context.Context, obj *global.StageSync) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		}}}
	}

	b := ctrl.NewControllerManagedBy(mgr).
		WatchesRawSource(
			source.Kind(
				r.RemoteCluster.GetCache(),
				&global.StageSync{},
				handler.TypedEnqueueRequestsFromMapFunc[*global.StageSync, reconcile.Request](mapStageSyncToRequests),
				predicate.TypedGenerationChangedPredicate[*global.StageSync]{},
			),
		)
	return b.
		Named("sync").
		Complete(r)
}
func (r *StageSyncReconciler) isStageVersionServed(ctx context.Context, logger logr.Logger, expectedVersion string) (bool, error) {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := r.LocalClient.Get(ctx, client.ObjectKey{Name: "stages.landscape.konfidence.cloud"}, crd); err != nil {
		return false, fmt.Errorf("unable to fetch Stage CRD: %w", err)
	}

	for _, v := range crd.Spec.Versions {
		logger.Info("served version of stage crd", "version", v.Name, "served", v.Served)
		if v.Name == expectedVersion && v.Served {
			return true, nil
		}
	}

	return false, nil
}

func (r *StageSyncReconciler) setAndPatchStatus(ctx context.Context, stageSync *global.StageSync, originalStageSync *global.StageSync, status metav1.ConditionStatus, reason, message string) error {
	meta.SetStatusCondition(&stageSync.Status.Conditions, metav1.Condition{
		Type:               global.StageSyncAppliedCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: stageSync.Generation,
		LastTransitionTime: metav1.Now(),
	})

	patch := client.MergeFrom(originalStageSync)
	if !reflect.DeepEqual(stageSync.Status, originalStageSync.Status) {
		if err := r.RemoteCluster.GetClient().Status().Patch(ctx, stageSync, patch); err != nil {
			return fmt.Errorf("unable to patch remote resource status: %w", err)
		}
	}

	return nil
}

func (r *StageSyncReconciler) falsifyAndPatchStatus(ctx context.Context, stageSync *global.StageSync, originalStageSync *global.StageSync, reason, message string) error {
	return r.setAndPatchStatus(ctx, stageSync, originalStageSync, metav1.ConditionFalse, reason, message)
}

// setStageDeletedCondition sets the StageDeleted condition on the remote StageSync
// and patches the status. It is used during the deletion lifecycle to report
// progress (e.g. deletion initiated, blocked, confirmed).
func (r *StageSyncReconciler) setStageDeletedCondition(ctx context.Context, stageSync, originalStageSync *global.StageSync, status metav1.ConditionStatus, reason, message string) error {
	meta.SetStatusCondition(&stageSync.Status.Conditions, metav1.Condition{
		Type:               global.StageDeletedCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: stageSync.Generation,
		LastTransitionTime: metav1.Now(),
	})
	patch := client.MergeFrom(originalStageSync)
	if !reflect.DeepEqual(stageSync.Status, originalStageSync.Status) {
		if err := r.RemoteCluster.GetClient().Status().Patch(ctx, stageSync, patch); err != nil {
			return fmt.Errorf("unable to patch StageDeleted condition: %w", err)
		}
	}
	return nil
}

// ensureStageSyncedByLabel patches a per-cluster label onto the remote
// StageSync object when it is absent. The label key is
// "synced-by-star/<cluster-name>" with value "true", so multiple star clusters
// can each add their own label without overwriting one another.
// If LandscapeName is empty the label is skipped to avoid writing an invalid key.
func (r *StageSyncReconciler) ensureStageSyncedByLabel(ctx context.Context, stageSync, originalStageSync *global.StageSync) error {
	if r.LandscapeName == "" {
		// LandscapeName is not configured — the synced-by label cannot be set.
		r.Recorder.Eventf(stageSync, nil, corev1.EventTypeWarning, "LandscapeNameNotConfigured", "EnsureStageSyncedByLabel",
			"LANDSCAPE_NAME is not set; skipping synced-by label on StageSync %s/%s", stageSync.Namespace, stageSync.Name)
		return nil
	}
	labelKey := stageSyncedByLabelPrefix + r.LandscapeName

	if stageSync.Labels[labelKey] == "true" {
		return nil
	}
	labels := stageSync.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[labelKey] = "true"
	stageSync.SetLabels(labels)
	patch := client.MergeFrom(originalStageSync)
	return r.RemoteCluster.GetClient().Patch(ctx, stageSync, patch)
}

// ensureFinalizer adds the sync controller finalizer to the remote StageSync
// if it is not already present.
//
// Parameters:
//   - stageSync: the current (possibly mutated) state of the remote StageSync object.
//   - originalStageSync: an unmodified deep-copy of stageSync used as the patch base.
//
// Return values:
//   - error: non-nil if adding the finalizer failed.
func (r *StageSyncReconciler) ensureFinalizer(ctx context.Context, stageSync, originalStageSync *global.StageSync) error {
	if controllerutil.ContainsFinalizer(stageSync, syncControllerFinalizer) {
		return nil
	}
	patch := client.MergeFrom(originalStageSync)
	controllerutil.AddFinalizer(stageSync, syncControllerFinalizer)
	if err := r.RemoteCluster.GetClient().Patch(ctx, stageSync, patch); err != nil {
		err = fmt.Errorf("unable to add finalizer to remote resource: %w", err)
		patchErr := r.falsifyAndPatchStatus(ctx, stageSync, originalStageSync, global.AddingFinalizerFailedReason, err.Error())
		return errors.Join(err, patchErr)
	}
	return nil
}

// handleStageSyncDeletion drives the deletion flow for a StageSync that is being
// deleted (DeletionTimestamp is set). It uses the local Stage object that was
// already fetched by the reconcile loop to avoid a redundant API call.
// If localStage.Name is empty the Stage was not found (NotFound was returned by the fetch).
//
// The method progresses through the following states:
//  1. Stage not found (localStage.Name == "") → remove the StageSync finalizer.
//     Returns ctrl.Result{} (no requeue) since the StageSync will be garbage collected.
//  2. Stage found with a DeletionTimestamp → blocked by finalizers; surface
//     StageDeletionBlocked and return RequeueAfter to check again later.
//  3. Stage found without a DeletionTimestamp → issue the delete, set
//     StageDeletionInitiated and return RequeueAfter to confirm removal.
func (r *StageSyncReconciler) handleStageSyncDeletion(ctx context.Context, stageSync, originalStageSync *global.StageSync, localStage *landscape.Stage) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(stageSync, syncControllerFinalizer) {
		// Finalizer already removed; nothing left to do.
		return ctrl.Result{}, nil
	}

	if localStage.Name == "" {
		// Stage is gone
		msg := fmt.Sprintf("stage for StageSync %s/%s successfully deleted", stageSync.Namespace, stageSync.Name)
		if err := r.setStageDeletedCondition(ctx, stageSync, originalStageSync, metav1.ConditionTrue, global.StageDeletionSuccessfulReason, msg); err != nil {
			return ctrl.Result{}, err
		}
		// Remove finalizer to allow StageSync to be garbage collected.
		patch := client.MergeFrom(originalStageSync)
		controllerutil.RemoveFinalizer(stageSync, syncControllerFinalizer)
		if err := r.RemoteCluster.GetClient().Patch(ctx, stageSync, patch); err != nil {
			err = fmt.Errorf("unable to remove finalizer from remote resource: %w", err)
			patchErr := r.falsifyAndPatchStatus(ctx, stageSync, originalStageSync, global.RemovingFinalizerFailedReason, err.Error())
			return ctrl.Result{}, errors.Join(err, patchErr)
		}
		// Finalizer removed – StageSync will be garbage collected; no requeue needed.
		return ctrl.Result{}, nil
	}

	// Stage still exists – check whether it is stuck on finalizers.
	if !localStage.DeletionTimestamp.IsZero() {
		// The Stage has a DeletionTimestamp but hasn't been removed yet → blocked by finalizers.
		remaining := localStage.GetFinalizers()
		msg := fmt.Sprintf("stage %s/%s deletion is blocked by finalizers: %v", localStage.Namespace, localStage.Name, remaining)
		r.Recorder.Eventf(stageSync, nil, corev1.EventTypeWarning, global.StageDeletionBlockedReason, "DeleteStage", msg)
		if err := r.setStageDeletedCondition(ctx, stageSync, originalStageSync, metav1.ConditionFalse, global.StageDeletionBlockedReason, msg); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: deletionRequeueInterval}, nil
	}

	// Stage exists and has no DeletionTimestamp → issue delete.
	if err := r.LocalClient.Delete(ctx, localStage); err != nil {
		msg := fmt.Sprintf("unable to delete local resource: %s", err)
		r.Recorder.Eventf(stageSync, nil, corev1.EventTypeWarning, global.StageDeletionFailedReason, "DeleteStage", msg)
		patchErr := r.falsifyAndPatchStatus(ctx, stageSync, originalStageSync, global.StageDeletionFailedReason, msg)
		return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
	}

	// Deletion issued – requeue to confirm removal.
	msg := fmt.Sprintf("stage %s/%s delete issued, waiting for confirmation", localStage.Namespace, localStage.Name)
	r.Recorder.Eventf(stageSync, nil, corev1.EventTypeNormal, global.StageDeletionInitiatedReason, "DeleteStage", msg)
	if err := r.setStageDeletedCondition(ctx, stageSync, originalStageSync, metav1.ConditionFalse, global.StageDeletionInitiatedReason, msg); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: deletionRequeueInterval}, nil
}

func getStageFromTemplate(stageTemplate runtime.RawExtension) (*landscape.Stage, error) {
	stage := &landscape.Stage{}
	if err := json.Unmarshal(stageTemplate.Raw, stage); err != nil {
		return nil, fmt.Errorf("unable to unmarshal stage template: %w", err)
	}
	log.Log.V(0).Info("unmarshalled stage template",
		"apiVersion", stage.APIVersion,
		"kind", stage.Kind,
		"name", stage.Name,
		"namespace", stage.Namespace,
	)
	if stage.Name == "" {
		return nil, fmt.Errorf("stage template is missing metadata.name")
	}
	if stage.Namespace == "" {
		return nil, fmt.Errorf("stage template is missing metadata.namespace")
	}
	if stage.APIVersion == "" {
		return nil, fmt.Errorf("stage template is missing apiVersion")
	}
	if stage.Kind == "" {
		return nil, fmt.Errorf("stage template is missing kind")
	}
	return stage, nil
}

func adjustStageFromTemplate(stage, stageTemplate *landscape.Stage, namespacedName types.NamespacedName) {
	stage.SetGroupVersionKind(stageTemplate.GroupVersionKind())
	stage.SetName(stageTemplate.Name)
	stage.SetNamespace(stageTemplate.Namespace)

	// set labels to identify the controller and the StageSync resource managing this stage
	stageLabels := stage.GetLabels()
	if stageLabels == nil {
		stageLabels = make(map[string]string)
	}
	stageLabels[managedByLabelKey] = StageSyncControllerName
	stageLabels[galaxyStageSyncLabelKey] = sanitizeLabelValue(namespacedName.String())
	stage.SetLabels(stageLabels)
}

// sanitizeLabelValue converts a string into a valid Kubernetes label value.
// It replaces '/' with '_' and lowercases the result to comply with the
// label value format: [a-z0-9.-_], max 63 characters.
func sanitizeLabelValue(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ToLower(s)
	if len(s) > 63 {
		s = s[:63]
	}
	return s
}

func reflectStageStatusOnStageSync(stage *landscape.Stage, stageSync *global.StageSync) error {
	raw, err := json.Marshal(stage.Status)
	if err != nil {
		return fmt.Errorf("unable to marshal stage status: %w", err)
	}
	stageSync.Status.StageStatus = runtime.RawExtension{Raw: raw}
	return nil
}
