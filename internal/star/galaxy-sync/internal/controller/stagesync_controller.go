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
	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
)

// StageSyncReconciler watches a StageSync object on the remote client (GCP) and creates/updates/deletes a corresponding Stage object on the local cluster (LCP).
type StageSyncReconciler struct {
	// LocalClient is the client accessing the LCP
	LocalClient client.Client
	// RemoteClient is the client accessing the GCP
	RemoteClient client.Client
	RemoteCache  cache.Cache
	Scheme       *runtime.Scheme
}

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stagesyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stagesyncs/status,verbs=get;update;patch

func (r *StageSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling StageSync")

	// fetch remote Resource
	remoteStageSync := &global.StageSync{}
	err := r.RemoteClient.Get(ctx, req.NamespacedName, remoteStageSync)
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
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, "InvalidStageTemplate", err.Error()) // TODO: add constant in stagesync_types.go
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
		err = fmt.Errorf("unable to verify if stage version is served: %w", err)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.StageCrdQueryFailedReason, err.Error())
		err = errors.Join(err, patchErr)
		return ctrl.Result{}, err
	}
	if !versionServed {
		err = fmt.Errorf("expected version %s is not present or not served in Stage CRD", stageTemplate.TypeMeta.GroupVersionKind().Version)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.APIVersionNotSupportedReason, err.Error())
		err = errors.Join(err, patchErr)
		return ctrl.Result{}, err
	}

	// check if namespace exists on local
	if err := r.LocalClient.Get(ctx, client.ObjectKey{Name: stageTemplate.Namespace}, &corev1.Namespace{}); err != nil {
		err = fmt.Errorf("unable to fetch local namespace %s: %w", req.Namespace, err)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.NamespaceNotFoundReason, err.Error())
		err = errors.Join(err, patchErr)
		return ctrl.Result{}, err
	}

	// fetch local resource
	localStage := &common.Stage{}
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
			err = fmt.Errorf("stage that is not managed by this StageSync resource already exists with desired name and namespace; expected label '%s: %s'", managedByLabelKey, getLabelValue(req.NamespacedName))
			patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.ConflictWithUnmanagedStageReason, err.Error())
			err = errors.Join(err, patchErr)
			return ctrl.Result{}, err
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
		err = fmt.Errorf("unable to create or update local resource: %w", err)
		patchErr := r.falsifyAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, global.StageCreationFailedReason, err.Error())
		err = errors.Join(err, patchErr)
		return ctrl.Result{}, err
	}

	// reflect status conditions of local on remote resource
	reflectStageStatusConditionsOnStageSync(localStage, remoteStageSync)

	// update status of remote resource
	msg := fmt.Sprintf("reconcile of local stage resource successful, operationsResult: %s", operationResult)
	logger.Info(msg)
	patchErr := r.setAndPatchStatus(ctx, remoteStageSync, originalRemoteStageSync, metav1.ConditionTrue, global.StageCreationSuccessfulReason, msg)
	if patchErr != nil {
		err = fmt.Errorf("unable to patch status of remote resource: %w", patchErr)
		return ctrl.Result{}, err
	}

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
				r.RemoteCache,
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
	if err := r.LocalClient.Get(ctx, client.ObjectKey{Name: "stages.common.konfidence.cloud"}, crd); err != nil {
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
		if err := r.RemoteClient.Status().Patch(ctx, stageSync, patch); err != nil {
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

func (r *StageSyncReconciler) handleFinalizer(ctx context.Context, stageSync, originalStageSync *global.StageSync, stage *common.Stage, stageFound bool) (bool, error) {
	// Check if StageSync is being deleted
	if stageSync.DeletionTimestamp.IsZero() {
		// Object is NOT being deleted
		if !controllerutil.ContainsFinalizer(stageSync, syncControllerFinalizer) {
			patch := client.MergeFrom(originalStageSync)
			controllerutil.AddFinalizer(stageSync, syncControllerFinalizer)
			if err := r.RemoteClient.Patch(ctx, stageSync, patch); err != nil {
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
				err = fmt.Errorf("unable to delete local resource: %w", err)
				patchErr := r.falsifyAndPatchStatus(ctx, stageSync, originalStageSync, global.StageDeletionFailedReason, err.Error())
				err = errors.Join(err, patchErr)
				return true, err
			}
		}
		// Remove finalizer
		patch := client.MergeFrom(originalStageSync)
		controllerutil.RemoveFinalizer(stageSync, syncControllerFinalizer)
		if err := r.RemoteClient.Patch(ctx, stageSync, patch); err != nil {
			err = fmt.Errorf("unable to remove finalizer from remote resource: %w", err)
			patchErr := r.falsifyAndPatchStatus(ctx, stageSync, originalStageSync, global.RemovingFinalizerFailedReason, err.Error())
			err = errors.Join(err, patchErr)
			return true, err
		}
	}
	return true, nil
}

func getStageFromTemplate(stageTemplate runtime.RawExtension) (*common.Stage, error) {
	stage := &common.Stage{}
	if err := json.Unmarshal(stageTemplate.Raw, stage); err != nil {
		log.Log.Error(err, "unable to unmarshal stage template")
		return nil, fmt.Errorf("unable to unmarshal stage template: %w", err)
	}
	return stage, nil
}

func adjustStageFromTemplate(stage, stageTemplate *common.Stage, namespacedName types.NamespacedName) {
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

func reflectStageStatusConditionsOnStageSync(stage *common.Stage, stageSync *global.StageSync) {
	for _, condition := range stage.Status.Conditions {
		meta.SetStatusCondition(&stageSync.Status.Conditions, metav1.Condition{
			Type:               condition.Type,
			Status:             condition.Status,
			Reason:             condition.Reason,
			Message:            condition.Message,
			ObservedGeneration: stageSync.Generation,
			LastTransitionTime: condition.LastTransitionTime,
		})
	}
}
