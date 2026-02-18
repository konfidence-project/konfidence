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
	"regexp"
	"strings"
	"time"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	"github.com/konfidence-project/gcp-stage-configuration-controller/pkg/ocm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"
)

const (
	DefaultReconcileInterval         = 30 * time.Second
	StageConfigurationControllerName = "stage-configuration-controller"
	ClusterMarker                    = "/clusters/"
	ClusterPattern                   = "\\/clusters\\/[^/]+$"
)

var (
	ClusterRegex = regexp.MustCompile(ClusterPattern)
)

// StageConfigurationReconciler reconciles a StageConfiguration object
type StageConfigurationReconciler struct {
	Mgr        mcmanager.Manager
	OCMClient  ocm.Client
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	SkipOci    bool
}

// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stageconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=global.konfidence.cloud,resources=stageconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.konfidence.cloud,resources=stages/status,verbs=get;update;patch

func (r *StageConfigurationReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stageConfiguration started...")
	log.Info(fmt.Sprintf("Cluster: %s", req.ClusterName))

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	clusterClient := cluster.GetClient()
	recorder := cluster.GetEventRecorder(StageConfigurationControllerName)

	// get stageConfiguration
	stageConfiguration := &global.StageConfiguration{}
	if err := clusterClient.Get(ctx, req.NamespacedName, stageConfiguration); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalStageConfiguration := stageConfiguration.DeepCopy()
	patch := client.MergeFrom(originalStageConfiguration)
	err = r.reconcileStageConfiguration(ctx, clusterClient, stageConfiguration, recorder)

	if !reflect.DeepEqual(stageConfiguration.Status, originalStageConfiguration.Status) {
		if patchError := clusterClient.Status().Patch(ctx, stageConfiguration, patch); patchError != nil {
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

func (r *StageConfigurationReconciler) reconcileStageConfiguration(ctx context.Context, clusterClient client.Client, stageConfiguration *global.StageConfiguration, recorder events.EventRecorder) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stageConfiguration")

	latestVersion := "1.0.0"
	if !r.SkipOci {
		// get latest vector version
		version, err := r.OCMClient.GetLatestComponentVersion(ctx, stageConfiguration.Spec.Vector)
		if err != nil {
			return fmt.Errorf("unable to get latest vector component version %s: %w", stageConfiguration.Spec.Vector, err)
		}

		latestVersion = version
	}

	// combine vector with latest version
	vector := stageConfiguration.Spec.Vector + ":" + latestVersion

	// write to cluster or workspace
	var targetClient client.Client
	if r.Mgr.GetProvider() == nil {
		targetClient = clusterClient
	} else {
		// copy the default kubeconfig and change the server url
		targetCfg := rest.CopyConfig(r.RestConfig)

		targetWorkspaceHost, err := r.getTargetWorkspaceHost(targetCfg.Host, stageConfiguration)
		if err != nil {
			return fmt.Errorf("unable to get target workspace host: %w", err)
		}

		targetCfg.Host = targetWorkspaceHost

		// create a new client for the target cluster
		cl, err := client.New(targetCfg, client.Options{
			Scheme: r.Scheme,
		})

		if err != nil {
			return fmt.Errorf("could not create target client from rest config %w", err)
		}

		targetClient = cl
	}

	// create or update stage
	stage, operationResult, err := r.createOrUpdateStage(ctx, targetClient, stageConfiguration, vector)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Stage %s %s with StageConfiguration %s", stage.Name, operationResult, stageConfiguration.Name)
	recorder.Eventf(stageConfiguration, nil, v1.EventTypeNormal, "StageConfigurationReconciled", "StageConfigurationReconciled", msg)
	log.Info(msg)
	return nil
}

func (r *StageConfigurationReconciler) createOrUpdateStage(ctx context.Context, targetClient client.Client, stageConfiguration *global.StageConfiguration, vector string) (*common.Stage, controllerutil.OperationResult, error) {
	stage := r.constructStage(stageConfiguration, vector)
	operationResult, err := controllerutil.CreateOrUpdate(ctx, targetClient, stage, func() error {
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
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
		Spec: common.StageSpec{
			Vector: vector,
		},
	}
}

func (r *StageConfigurationReconciler) getTargetWorkspaceHost(host string, stageConfiguration *global.StageConfiguration) (string, error) {
	// if kcp is used the target workspace is mandatory
	if stageConfiguration.Spec.TargetWorkspace == nil || len(*stageConfiguration.Spec.TargetWorkspace) == 0 {
		return "", fmt.Errorf("stage configuration does not contain a target workspace")
	}

	if !ClusterRegex.MatchString(host) {
		return "", fmt.Errorf("could not match clusters entry at end of config host %s", host)
	}

	separatorIdx := strings.LastIndex(host, ClusterMarker)
	if separatorIdx == -1 {
		return "", fmt.Errorf("missing clusters entry in config host %s", host)
	}

	return host[:separatorIdx] + ClusterMarker + *stageConfiguration.Spec.TargetWorkspace, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageConfigurationReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		For(&global.StageConfiguration{}).
		Named(StageConfigurationControllerName).
		Complete(r)
}
