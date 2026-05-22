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

package stageconfiguration

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	landscape "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"

	"github.com/konfidence-project/konfidence/internal/galaxy/stage-configuration/internal/ocm"
	"github.com/konfidence-project/konfidence/internal/galaxy/stage-configuration/internal/ports"
	"github.com/konfidence-project/konfidence/internal/galaxy/stage-configuration/internal/template"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

const (
	defaultReconcileInterval         = 30 * time.Second
	stageConfigurationControllerName = "stage-configuration-controller"
	clusterMarker                    = "/clusters/"
	clusterPattern                   = "\\/clusters\\/[^/]+$"
)

var clusterRegex = regexp.MustCompile(clusterPattern)

// StageConfigurationReconciler reconciles a StageConfiguration object
type StageConfigurationReconciler struct {
	Mgr                mcmanager.Manager
	OcmClientProvider  repository.ClientProvider
	VectorPortProvider ports.VectorPortProvider
	Scheme             *runtime.Scheme
	RestConfig         *rest.Config
	OcmVerifier        crypto.Verifier
}

func NewStageConfigurationReconciler(
	mgr mcmanager.Manager,
	scheme *runtime.Scheme,
	restConfig *rest.Config,
	vectorVerifier crypto.Verifier,
) *StageConfigurationReconciler {
	return &StageConfigurationReconciler{
		Mgr:                mgr,
		OcmClientProvider:  repository.DefaultOciClientProvider,
		VectorPortProvider: ocm.DefaultPortProvider,
		Scheme:             scheme,
		RestConfig:         restConfig,
		OcmVerifier:        vectorVerifier,
	}
}

// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=stageconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=stageconfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=stagesyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=galaxy.konfidence.cloud,resources=stagesyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *StageConfigurationReconciler) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile stageConfiguration started...")
	log.Info(fmt.Sprintf("Cluster: %s", req.ClusterName))

	cluster, err := r.Mgr.GetCluster(ctx, req.ClusterName)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	clusterClient := cluster.GetClient()
	recorder := cluster.GetEventRecorder(stageConfigurationControllerName)

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
	return ctrl.Result{RequeueAfter: defaultReconcileInterval}, nil
}

func (r *StageConfigurationReconciler) reconcileStageConfiguration(
	ctx context.Context,
	clusterClient client.Client,
	stageConfiguration *global.StageConfiguration,
	recorder events.EventRecorder,
) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling stageConfiguration")
	r.updateStageConfigurationReadyStatus(stageConfiguration, false, "")

	ocmClient, err := r.OcmClientProvider.NewClient(ctx, clusterClient, stageConfiguration.GetNamespace(), stageConfiguration.Spec.Config)
	if err != nil {
		return fmt.Errorf("unable to create OCM client: %w", err)
	}

	vectorPort := r.VectorPortProvider.NewVectorPort(r.OcmVerifier, ocmClient)

	// get vector with specific version or alias
	vector, err := vectorPort.GetLatestVectorVersion(ctx, stageConfiguration.Spec.Vector)
	if err != nil {
		return fmt.Errorf("unable to get vector component version %s: %w", stageConfiguration.Spec.Vector, err)
	}

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

	// create or update stageSync
	stageSync, operationResult, err := r.createOrUpdateStageSync(ctx, targetClient, stageConfiguration, vector)
	if err != nil {
		return err
	}

	// mark stageConfiguration as ready
	r.updateStageConfigurationReadyStatus(stageConfiguration, true, fmt.Sprintf("StageConfiguration %s reconciled", stageConfiguration.Name))

	msg := fmt.Sprintf("StageSync %s %s with StageConfiguration %s", stageSync.Name, operationResult, stageConfiguration.Name)
	recorder.Eventf(stageConfiguration, nil, v1.EventTypeNormal, "StageConfigurationReconciled", "StageConfigurationReconciled", msg)
	log.Info(msg)
	return nil
}

func (r *StageConfigurationReconciler) createOrUpdateStageSync(
	ctx context.Context, targetClient client.Client,
	stageConfiguration *global.StageConfiguration, vector string,
) (*global.StageSync, controllerutil.OperationResult, error) {
	stageSync, stageTemplateBytes, err := r.constructStageSync(stageConfiguration, vector)
	if err != nil {
		return nil, controllerutil.OperationResultNone, err
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, targetClient, stageSync, func() error {
		var originalTemplate, stageTemplate template.StageTemplate
		if err := json.Unmarshal(stageSync.Spec.StageTemplate.Raw, &originalTemplate); err != nil {
			return err
		}
		if err := json.Unmarshal(stageTemplateBytes, &stageTemplate); err != nil {
			return err
		}

		if !reflect.DeepEqual(originalTemplate, stageTemplate) {
			stageSync.Spec.StageTemplate.Raw = stageTemplateBytes
		}

		return nil
	})
	if err != nil {
		return nil, operationResult, fmt.Errorf("failed to create or update stageSync: %w", err)
	}
	return stageSync, operationResult, nil
}

func (r *StageConfigurationReconciler) constructStageSync(stageConfiguration *global.StageConfiguration, vector string) (*global.StageSync, []byte, error) {
	stageTemplate := r.constructStageTemplate(stageConfiguration, vector)
	stageTemplateJSON, err := json.Marshal(stageTemplate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal stage template: %w", err)
	}

	// TODO this might lead to naming collisions. to resolve this issue one
	// TODO could e.g. use a digest that is computed using stage name and tenant
	stageSyncName := fmt.Sprintf("sync-%s", stageConfiguration.Spec.Name)

	return &global.StageSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stageSyncName,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
		Spec: global.StageSyncSpec{
			StageTemplate: runtime.RawExtension{Raw: stageTemplateJSON},
		},
	}, stageTemplateJSON, nil
}

func (r *StageConfigurationReconciler) constructStageTemplate(stageConfiguration *global.StageConfiguration, vector string) *template.StageTemplate {
	// TODO replace APIVersion with a configured or determined value
	return &template.StageTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       landscape.StageKind,
			APIVersion: "star.konfidence.cloud/v1alpha1",
		},
		Metadata: template.NamespacedName{
			Name:      stageConfiguration.Spec.Name,
			Namespace: stageConfiguration.Spec.TargetNamespace,
		},
		Spec: landscape.StageSpec{
			Vector: vector,
		},
	}
}

func (r *StageConfigurationReconciler) getTargetWorkspaceHost(host string, stageConfiguration *global.StageConfiguration) (string, error) {
	// if kcp is used the target workspace is mandatory
	if stageConfiguration.Spec.TargetWorkspace == nil || len(*stageConfiguration.Spec.TargetWorkspace) == 0 {
		return "", fmt.Errorf("stage configuration does not contain a target workspace")
	}

	if !clusterRegex.MatchString(host) {
		return "", fmt.Errorf("could not match clusters entry at end of config host %s", host)
	}

	separatorIdx := strings.LastIndex(host, clusterMarker)
	if separatorIdx == -1 {
		return "", fmt.Errorf("missing clusters entry in config host %s", host)
	}

	return host[:separatorIdx] + clusterMarker + *stageConfiguration.Spec.TargetWorkspace, nil
}

func (r *StageConfigurationReconciler) updateStageConfigurationReadyStatus(stageConfiguration *global.StageConfiguration, ready bool, message string) {
	status := metav1.ConditionFalse
	if ready {
		status = metav1.ConditionTrue
	}

	meta.SetStatusCondition(&stageConfiguration.Status.Conditions, metav1.Condition{
		Type:               global.StageConfigurationReadyCondition,
		Status:             status,
		Reason:             global.StageConfigurationReadyCondition,
		Message:            message,
		ObservedGeneration: stageConfiguration.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *StageConfigurationReconciler) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		For(&global.StageConfiguration{}, mcbuilder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named(stageConfigurationControllerName).
		Complete(r)
}
