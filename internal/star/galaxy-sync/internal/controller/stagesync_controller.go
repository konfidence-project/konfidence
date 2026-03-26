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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	syncControllerFinalizer  = "konfidence.cloud/stage-sync-finalizer"
	defaultReconcileInterval = 30 * time.Second
	managedByLabelKey        = "managed-by"
	StageSyncControllerName  = "stage-sync-controller"
)

// StageSyncReconciler watches a StageSync object on the remote client (GCP) and creates/updates/deletes a corresponding Stage object on the local cluster (LCP).
type StageSyncReconciler struct {
	// LocalClient is the client accessing the LCP
	LocalClient client.Client
	// RemoteCluster is the cluster accessor for the GCP side. Its cache is
	// managed by the controller-runtime manager, which guarantees it is
	// started (and synced) before any informers sourced from it.
	RemoteCluster cluster.Cluster
	Scheme        *runtime.Scheme
	Recorder      events.EventRecorder
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
	stageFound := !apierrors.IsNotFound(err)
	if stageFound {
		// check if existing stage is managed by this StageSync resource
		namespacedNameString, ok := localStage.GetLabels()[managedByLabelKey]
		if !ok || namespacedNameString != getLabelValue(req.NamespacedName) {
			msg := fmt.Sprintf("stage that is not managed by this StageSync resource already exists with desired name and namespace; expected label '%s: %s'", managedByLabelKey, getLabelValue(req.NamespacedName))
			logger.Error(nil, msg)
			r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeWarning, global.ConflictWithUnmanagedStageReason, "CheckManagedStage", msg)
			patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.ConflictWithUnmanagedStageReason, msg)
			return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
		}
	}

	// handle finalizer to control deletion
	isBeingDeleted, err := r.handleFinalizer(ctx, remoteStageSync, originalRemoteStageSync, localStage, stageFound)
	if err != nil {
		return ctrl.Result{}, err
	}
	if isBeingDeleted {
		return ctrl.Result{}, nil
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
		r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeWarning, global.StageCreationFailedReason, "CreateOrUpdateStage", msg)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.StageCreationFailedReason, msg)
		return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
	}

	// reflect status of local stage on remote resource
	if err := reflectStageStatusOnStageSync(localStage, remoteStageSync); err != nil {
		msg := fmt.Sprintf("unable to reflect stage status on StageSync: %s", err)
		logger.Error(err, msg)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.StageCreationFailedReason, msg)
		return ctrl.Result{}, errors.Join(errors.New(msg), patchErr)
	}

	// update status of remote resource
	reconcileMsg := fmt.Sprintf("reconcile of local stage resource successful, operationResult: %s", operationResult)
	patchErr := r.setAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, metav1.ConditionTrue, global.StageCreationSuccessfulReason, reconcileMsg)
	if patchErr != nil {
		err = fmt.Errorf("unable to patch status of remote resource: %w", patchErr)
		return ctrl.Result{}, err
	}

	successMsg := fmt.Sprintf("stage %s/%s reconciled successfully (operation: %s)", localStage.Namespace, localStage.Name, operationResult)
	logger.Info(successMsg)
	r.Recorder.Eventf(remoteStageSync, nil, corev1.EventTypeNormal, global.StageCreationSuccessfulReason, "CreateOrUpdateStage", successMsg)

	return ctrl.Result{RequeueAfter: reconcileInterval.Duration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapStageSyncToRequests := func(ctx context.Context, obj *global.StageSync) []reconcile.Request {
		return r.getNamespacedNameReconcileRequest(ctx, obj)
	}

	b := ctrl.NewControllerManagedBy(mgr).
		WatchesRawSource(
			source.Kind(
				r.RemoteCluster.GetCache(),
				&global.StageSync{},
				handler.TypedEnqueueRequestsFromMapFunc[*global.StageSync, reconcile.Request](mapStageSyncToRequests),
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

func (r *StageSyncReconciler) getNamespacedNameReconcileRequest(_ context.Context, object client.Object) []reconcile.Request {
	return []reconcile.Request{{NamespacedName: types.NamespacedName{
		Name:      object.GetName(),
		Namespace: object.GetNamespace(),
	}}}
}

func (r *StageSyncReconciler) handleFinalizer(ctx context.Context, stageSync, originalStageSync *global.StageSync, stage *landscape.Stage, stageFound bool) (bool, error) {
	// Check if StageSync is being deleted
	if stageSync.DeletionTimestamp.IsZero() {
		// Object is NOT being deleted
		if !controllerutil.ContainsFinalizer(stageSync, syncControllerFinalizer) {
			patch := client.MergeFrom(originalStageSync)
			controllerutil.AddFinalizer(stageSync, syncControllerFinalizer)
			if err := r.RemoteCluster.GetClient().Patch(ctx, stageSync, patch); err != nil {
				err = fmt.Errorf("unable to add finalizer to remote resource: %w", err)
				patchErr := r.falsifyAndPatchStatus(ctx, stageSync, originalStageSync, global.AddingFinalizerFailedReason, err.Error())
				err = errors.Join(err, patchErr)
				return false, err
			}
		}
		return false, nil
	}
	// Object IS being deleted
	if controllerutil.ContainsFinalizer(stageSync, syncControllerFinalizer) {
		if stageFound {
			// Delete corresponding local resource if it exists
			if err := r.LocalClient.Delete(ctx, stage); err != nil {
				msg := fmt.Sprintf("unable to delete local resource: %s", err)
				r.Recorder.Eventf(stageSync, nil, corev1.EventTypeWarning, global.StageDeletionFailedReason, "DeleteStage", msg)
				patchErr := r.falsifyAndPatchStatus(ctx, stageSync, originalStageSync, global.StageDeletionFailedReason, msg)
				return true, errors.Join(errors.New(msg), patchErr)
			}
			successMsg := fmt.Sprintf("stage %s/%s deleted successfully", stage.Namespace, stage.Name)
			r.Recorder.Eventf(stageSync, nil, corev1.EventTypeNormal, global.StageDeletionFailedReason, "DeleteStage", successMsg)
		}
		// Remove finalizer
		patch := client.MergeFrom(originalStageSync)
		controllerutil.RemoveFinalizer(stageSync, syncControllerFinalizer)
		if err := r.RemoteCluster.GetClient().Patch(ctx, stageSync, patch); err != nil {
			err = fmt.Errorf("unable to remove finalizer from remote resource: %w", err)
			patchErr := r.falsifyAndPatchStatus(ctx, stageSync, originalStageSync, global.RemovingFinalizerFailedReason, err.Error())
			err = errors.Join(err, patchErr)
			return true, err
		}
	}
	return true, nil
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

	// set label to identify the StageSync resource managing this stage
	stageLabels := stage.GetLabels()
	if stageLabels == nil {
		stageLabels = make(map[string]string)
	}
	stageLabels[managedByLabelKey] = getLabelValue(namespacedName)
	stage.SetLabels(stageLabels)
}

func getLabelValue(namespacedName types.NamespacedName) string {
	return strings.ReplaceAll(namespacedName.String(), "/", "_")
}

func reflectStageStatusOnStageSync(stage *landscape.Stage, stageSync *global.StageSync) error {
	raw, err := json.Marshal(stage.Status)
	if err != nil {
		return fmt.Errorf("unable to marshal stage status: %w", err)
	}
	stageSync.Status.StageStatus = runtime.RawExtension{Raw: raw}
	return nil
}
