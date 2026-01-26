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
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	"github.com/konfidence-project/gcp-stage-configuration-controller/pkg/ocm"
)

const (
	DefaultReconcileInterval         = 30 * time.Second
	StageConfigurationControllerName = "stage-configuration-controller"
)

// StageConfigurationReconciler reconciles a StageConfiguration object
type StageConfigurationReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	OCMClient ocm.Client
	Recorder  record.EventRecorder
}

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stageconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stageconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages/status,verbs=get;update;patch

func (r *StageConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stageConfiguration started...")

	// get stageConfiguration
	stageConfiguration := &global.StageConfiguration{}
	if err := r.Get(ctx, req.NamespacedName, stageConfiguration); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStageConfiguration := stageConfiguration.DeepCopy()
	patch := client.MergeFrom(originalStageConfiguration)
	err := r.reconcileStageConfiguration(ctx, stageConfiguration)

	if !reflect.DeepEqual(stageConfiguration.Status, originalStageConfiguration.Status) {
		if patchError := r.Client.Status().Patch(ctx, stageConfiguration, patch); patchError != nil {
			patchErrorMessage := "unable to update stageConfiguration status"

			if err != nil {
				reconcileError := fmt.Errorf("an error occurred while reconciling stageConfiguration: %w", err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}

	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: DefaultReconcileInterval}, nil
}

func (r *StageConfigurationReconciler) reconcileStageConfiguration(ctx context.Context, stageConfiguration *global.StageConfiguration) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stageConfiguration")

	// get latest vector version
	latestVersion, err := r.OCMClient.GetLatestComponentVersion(ctx, stageConfiguration.Spec.Vector)
	if err != nil {
		return fmt.Errorf("unable to get latest vector component version %s: %w", stageConfiguration.Spec.Vector, err)
	}

	// combine vector with latest version
	vector := stageConfiguration.Spec.Vector + ":" + latestVersion

	// create or update stage
	stage, operationResult, err := r.createOrUpdateStage(ctx, stageConfiguration, vector)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Stage %s %s with StageConfiguration %s", stage.Name, operationResult, stageConfiguration.Name)
	r.Recorder.Event(stageConfiguration, v1.EventTypeNormal, "StageConfigurationReconciled", msg)
	log.Info(msg)
	return nil
}

func (r *StageConfigurationReconciler) createOrUpdateStage(ctx context.Context, stageConfiguration *global.StageConfiguration, vector string) (*common.Stage, controllerutil.OperationResult, error) {
	stage := r.constructStage(stageConfiguration, vector)
	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, stage, func() error {
		stage.Spec.Vector = vector
		return nil
	})

	if err != nil {
		return nil, operationResult, fmt.Errorf("failed to create or update stage: %w", err)
	}

	return stage, operationResult, nil
}

func (r *StageConfigurationReconciler) constructStage(stageConfiguration *global.StageConfiguration, vector string) *common.Stage {
	return &common.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageConfiguration.Spec.Name,
			Namespace: stageConfiguration.Namespace,
		},
		Spec: common.StageSpec{
			Vector: vector,
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&global.StageConfiguration{}).
		Named(StageConfigurationControllerName).
		Complete(r)
}
