/*
Copyright 2025.

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
	"strings"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	"github.com/konfidence-project/landscape-task-orchestration-controller/internal/graph"
	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	layerSucceeded          = 0
	layerPartiallySucceeded = 1
	layerPending            = 2
)

// TaskOrchestrationReconciler reconciles a VectorMigration object
type TaskOrchestrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type MapTasksResult struct {
	taskExecutionsByName           map[string]landscape.TaskExecution
	successfulTaskExecutionsByName map[string]bool
	taskFailed                     bool
}

// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectormigrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=vectormigrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversionusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=stageversionusages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=artifactdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=artifactdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=taskexecutions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.cloud,resources=taskexecutions/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *TaskOrchestrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile started...")

	// get vectorMigration
	vectorMigration := &landscape.VectorMigration{}
	if err := r.Get(ctx, req.NamespacedName, vectorMigration); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, landscape.VectorMigrationSucceeded) {
		return r.cleanupVectorMigration(ctx, req, vectorMigration)
	}

	originalVectorMigration := vectorMigration.DeepCopy()
	patch := client.MergeFrom(originalVectorMigration)
	err := r.reconcileVectorMigration(ctx, req, vectorMigration)

	if !reflect.DeepEqual(vectorMigration.Status, originalVectorMigration.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorMigration, patch); patchError != nil {
			patchErrorMessage := "unable to update vectorMigration status"

			if err != nil {
				reconcileError := fmt.Errorf("an error occurred while reconciling vectorMigration: %w", err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}

	return ctrl.Result{}, err
}

func (r *TaskOrchestrationReconciler) reconcileVectorMigration(ctx context.Context, req ctrl.Request, vectorMigration *landscape.VectorMigration) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling vectorMigration")

	// check if a stageVersionUsage already exists
	stageVersionUsage, err := r.createOrGetStageVersionUsage(ctx, req, vectorMigration)
	if err != nil {
		return err
	}

	// get vectorDeployment
	vectorDeployment, err := r.getVectorDeployment(ctx, vectorMigration)
	if err != nil {
		return fmt.Errorf("unable to fetch vectorDeployment: %w", err)
	}

	// get all tasks
	tasks, err := r.getArtifactDeploymentsAndTasks(ctx, vectorDeployment, vectorMigration)
	if err != nil {
		return fmt.Errorf("unable to fetch artifactDeployments and tasks: %w", err)
	}

	// sort tasks
	_, layers, err := graph.SortTasks(tasks)
	if err != nil {
		return fmt.Errorf("unable to sort task dependency graph: %w", err)
	}

	// map taskExecutions by name
	mappedTasks, err := r.mapTasks(ctx, req)
	if err != nil {
		return fmt.Errorf("mapping taskExecutions failed: %w", err)
	}

	if mappedTasks.taskFailed {
		// if at least one of the tasks failed mark vectorMigration as failed
		meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{Type: landscape.VectorMigrationFailed,
			Status: metav1.ConditionTrue, Reason: landscape.VectorMigrationFailed,
			Message: fmt.Sprintf("Reconciling VectorMigration %s failed", vectorMigration.Name)})
		return fmt.Errorf("reconciling VectorMigration failed")
	}

	// process all layers
	for _, layer := range layers {
		status, err := r.processTaskLayer(ctx, vectorMigration, layer, mappedTasks.taskExecutionsByName, mappedTasks.successfulTaskExecutionsByName)
		if err != nil {
			return fmt.Errorf("failed to process tasks: %w", err)
		}

		if status == layerPending {
			log.Info("Task layer still pending... retry later")
			// wait for taskExecution status change notifications
			return nil
		}
	}

	log.Info("Finished processing tasks")

	// mark vectorMigration as successful
	meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{Type: landscape.VectorMigrationSucceeded,
		Status: metav1.ConditionTrue, Reason: landscape.VectorMigrationSucceeded,
		Message: fmt.Sprintf("Successfully reconciled VectorMigration %s", vectorMigration.Name)})

	log.Info("Cleaning up resources...")

	// delete taskExecutions
	if err := r.deleteTaskExecutions(ctx, maps.Values(mappedTasks.taskExecutionsByName)); err != nil {
		return fmt.Errorf("unable to delete taskExecutions: %w", err)
	}

	// delete stageVersionUsage
	if err := r.Delete(ctx, stageVersionUsage); err != nil {
		return fmt.Errorf("unable to delete stageVersionUsage for vectorMigration: %w", err)
	}

	log.Info("VectorMigration reconciled")
	return nil
}

func (r *TaskOrchestrationReconciler) getVectorDeployment(ctx context.Context, vectorMigration *landscape.VectorMigration) (*landscape.VectorDeployment, error) {
	adaptedVectorName, err := adaptVectorName(vectorMigration.Spec.Vector)
	if err != nil {
		return nil, err
	}

	vectorDeployment := &landscape.VectorDeployment{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: vectorMigration.Namespace,
		Name:      adaptedVectorName,
	}, vectorDeployment)

	return vectorDeployment, err
}

func (r *TaskOrchestrationReconciler) getArtifactDeployment(ctx context.Context, vectorMigration *landscape.VectorMigration, name string) (*landscape.ArtifactDeployment, error) {
	artifactDeployment := &landscape.ArtifactDeployment{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: vectorMigration.Namespace,
		Name:      name,
	}, artifactDeployment)

	return artifactDeployment, err
}

func (r *TaskOrchestrationReconciler) getTaskExecutions(ctx context.Context, req ctrl.Request) ([]landscape.TaskExecution, error) {
	// get all taskExecutions that are owned by this vectorMigration
	taskExecutions := &landscape.TaskExecutionList{}
	if err := r.List(ctx, taskExecutions, client.InNamespace(req.Namespace), client.MatchingFields{vectorMigrationOwnerKey: req.Name}); err != nil {
		return nil, fmt.Errorf("unable to list taskExecutions: %w", err)
	}

	return taskExecutions.Items, nil
}

func (r *TaskOrchestrationReconciler) deleteTaskExecutions(ctx context.Context, taskExecutions []landscape.TaskExecution) error {
	for _, taskExecution := range taskExecutions {
		if err := r.Delete(ctx, &taskExecution); err != nil {
			return fmt.Errorf("unable to delete taskExecution for vectorMigration: %w", err)
		}
	}

	return nil
}

func (r *TaskOrchestrationReconciler) createOrGetStageVersionUsage(ctx context.Context, req ctrl.Request, vectorMigration *landscape.VectorMigration) (*landscape.StageVersionUsage, error) {
	log := logf.FromContext(ctx)
	stageVersionUsage := &landscape.StageVersionUsage{}
	err := r.Get(ctx, types.NamespacedName{Name: getStageVersionUsageName(vectorMigration.Spec.StageVersion), Namespace: req.Namespace}, stageVersionUsage)
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("unable to fetch stageVersionUsage: %w", err)
	}
	if err != nil && errors.IsNotFound(err) {
		// create new stageVersionUsage
		stageVersionUsage, err = r.constructStageVersionUsage(vectorMigration)
		if err != nil {
			return nil, fmt.Errorf("unable to construct vectorDeployment from template: %w", err)
		}

		if err := r.Create(ctx, stageVersionUsage); err != nil {
			return nil, fmt.Errorf("unable to create stageVersionUsage: %w", err)
		}

		log.V(1).Info("Created stageVersionUsage", "stageVersionUsage", stageVersionUsage)
	}

	// check if stageVersionUsage has vectorMigration owner ref
	hasRef, err := controllerutil.HasOwnerReference(stageVersionUsage.OwnerReferences, vectorMigration, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("unable to check stageVersionUsage owner reference: %w", err)
	}

	if !hasRef {
		log.Info("Set vectorMigration owner for stageVersionUsage")

		// set vectorMigration as owner of the stageVersionUsage
		if err := controllerutil.SetOwnerReference(vectorMigration, stageVersionUsage, r.Scheme); err != nil {
			return nil, fmt.Errorf("unable to set owner reference for stage version usage: %w", err)
		}

		if err := r.Update(ctx, stageVersionUsage); err != nil {
			return nil, fmt.Errorf("failed to update stageVersionUsage owner references: %w", err)
		}
	}

	// get stageVersion
	stageVersion := &landscape.StageVersion{}
	if err := r.Get(ctx, types.NamespacedName{Name: vectorMigration.Spec.StageVersion, Namespace: req.Namespace}, stageVersion); err != nil {
		return nil, fmt.Errorf("could not get stageVersion: %w", err)
	}

	hasRef, err = controllerutil.HasOwnerReference(stageVersion.OwnerReferences, stageVersionUsage, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("unable to check stageVersion owner reference: %w", err)
	}

	if hasRef {
		return stageVersionUsage, nil
	}

	log.Info("Set stageVersionUsage owner for stageVersion")

	// set stageVersionUsage as owner of the stageVersion
	if err := controllerutil.SetOwnerReference(stageVersionUsage, stageVersion, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to add stageVersionUsage ownerRef to stageVersion: %w", err)
	}

	if err := r.Update(ctx, stageVersion); err != nil {
		return nil, fmt.Errorf("failed to update stageVersion owner references: %w", err)
	}

	log.Info("Successfully set stageVersionUsage as owner of stageVersion")
	return stageVersionUsage, nil
}

func (r *TaskOrchestrationReconciler) getArtifactDeploymentsAndTasks(ctx context.Context, vectorDeployment *landscape.VectorDeployment, vectorMigration *landscape.VectorMigration) ([]landscape.TaskManifest, error) {
	artifactDeployments := make(map[string]landscape.ArtifactDeployment)
	numberOfTasks := 0
	for _, deploymentReference := range vectorDeployment.Status.ResultingArtifactDeployments {
		if landscape.ArtifactDeploymentKind != deploymentReference.Kind {
			return nil, fmt.Errorf("unable to parse artifactDeployment. Invalid kind: %s", deploymentReference.Kind)
		}

		// get artifactDeployment
		artifactDeployment, err := r.getArtifactDeployment(ctx, vectorMigration, deploymentReference.Name)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch artifactDeployment: %w", err)
		}

		artifactDeployments[deploymentReference.Name] = *artifactDeployment
		numberOfTasks = numberOfTasks + len(artifactDeployment.Spec.TaskManifests)
	}

	// get all task manifests from deployments
	tasks := make([]landscape.TaskManifest, 0, numberOfTasks)
	for _, artifactDeployment := range artifactDeployments {
		tasks = append(tasks, artifactDeployment.Spec.TaskManifests...)
	}

	return tasks, nil
}

func (r *TaskOrchestrationReconciler) mapTasks(ctx context.Context, req ctrl.Request) (*MapTasksResult, error) {
	log := logf.FromContext(ctx)
	taskExecutions, err := r.getTaskExecutions(ctx, req)
	if err != nil {
		return nil, err
	}

	// map taskExecutions by name
	mappedTasks := &MapTasksResult{}
	taskExecutionsByName := make(map[string]landscape.TaskExecution)
	successfulTaskExecutionsByName := make(map[string]bool)
	for _, taskExecution := range taskExecutions {
		if taskExecutionFailed(taskExecution) {
			log.Info("Task execution failed", "taskExecution", taskExecution)
			mappedTasks.taskFailed = true
			return mappedTasks, nil
		}

		taskExecutionsByName[taskExecution.Name] = taskExecution
		successfulTaskExecutionsByName[taskExecution.Name] = taskExecutionSucceeded(taskExecution)
	}

	mappedTasks.taskExecutionsByName = taskExecutionsByName
	mappedTasks.successfulTaskExecutionsByName = successfulTaskExecutionsByName
	return mappedTasks, nil
}

func (r *TaskOrchestrationReconciler) processTaskLayer(ctx context.Context, vectorMigration *landscape.VectorMigration, layer []landscape.TaskManifest, taskExecutionsByName map[string]landscape.TaskExecution, successfulTaskExecutionsByName map[string]bool) (int, error) {
	status := layerPending
	succeeded := 0

	for _, task := range layer {
		if _, ok := taskExecutionsByName[task.Name]; !ok {
			// create taskExecution if it does not exist and all dependencies have already been successfully processed
			if allTaskDependenciesSucceeded(task, successfulTaskExecutionsByName) {
				taskExecution, err := r.constructTaskExecution(vectorMigration, task, vectorMigration.Namespace)

				if err != nil {
					return status, fmt.Errorf("unable to construct taskExecution from template: %w", err)
				}

				if err := r.Create(ctx, taskExecution); err != nil {
					return status, fmt.Errorf("unable to create taskExecution: %w", err)
				}
			}
		}

		if successfulTaskExecutionsByName[task.Name] {
			succeeded++
		}
	}

	if succeeded == len(layer) {
		status = layerSucceeded
	} else if succeeded > 0 {
		status = layerPartiallySucceeded
	}

	return status, nil
}

var (
	vectorMigrationOwnerKey = ".metadata.controller"
	apiGVStr                = landscape.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *TaskOrchestrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &landscape.TaskExecution{}, vectorMigrationOwnerKey, func(rawObj client.Object) []string {
		// grab the taskExecution object and extract the owner
		taskExecution := rawObj.(*landscape.TaskExecution)
		owner := metav1.GetControllerOf(taskExecution)
		if owner == nil {
			return nil
		}
		// make sure it is a stage...
		if owner.APIVersion != apiGVStr || owner.Kind != landscape.VectorMigrationKind {
			return nil
		}

		// and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return fmt.Errorf("unable to create index for owner reference of task execution: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorMigration{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&landscape.TaskExecution{}).
		Named("taskOrchestration").
		Complete(r)
}

func (r *TaskOrchestrationReconciler) constructStageVersionUsage(vectorMigration *landscape.VectorMigration) (*landscape.StageVersionUsage, error) {
	stageVersionUsage := &landscape.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getStageVersionUsageName(vectorMigration.Spec.StageVersion),
			Namespace: vectorMigration.Namespace,
		},
		Spec: landscape.StageVersionUsageSpec{
			Reason: "VectorMigration",
			StageVersionRef: &landscape.StageVersionRef{
				Name: vectorMigration.Spec.StageVersion,
			},
		},
	}

	// set vectorMigration as owner of the stageVersionUsage
	if err := controllerutil.SetOwnerReference(vectorMigration, stageVersionUsage, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set owner reference for stage version usage: %w", err)
	}

	return stageVersionUsage, nil
}

func (r *TaskOrchestrationReconciler) constructTaskExecution(vectorMigration *landscape.VectorMigration, taskManifest landscape.TaskManifest, namespace string) (*landscape.TaskExecution, error) {
	taskExecution := &landscape.TaskExecution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskManifest.Name,
			Namespace: namespace,
		},
		Spec: landscape.TaskExecutionSpec(taskManifest),
	}

	// set vectorMigration as controller
	if err := ctrl.SetControllerReference(vectorMigration, taskExecution, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for taskExecution: %w", err)
	}
	return taskExecution, nil
}

func (r *TaskOrchestrationReconciler) cleanupVectorMigration(ctx context.Context, req ctrl.Request, vectorMigration *landscape.VectorMigration) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Cleanup vector migration")

	// get all taskExecutions for this vectorMigration
	taskExecutions, err := r.getTaskExecutions(ctx, req)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not get task executions: %w", err)
	}

	// and delete tasks if necessary
	if err := r.deleteTaskExecutions(ctx, taskExecutions); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete task executions: %w", err)
	}

	// check if stageVersionUsage still exists and should be deleted
	stageVersionUsage := &landscape.StageVersionUsage{}
	if err := r.Get(ctx, types.NamespacedName{Name: getStageVersionUsageName(vectorMigration.Spec.StageVersion), Namespace: req.Namespace}, stageVersionUsage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.Delete(ctx, stageVersionUsage); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to delete stageVersionUsage for vectorMigration: %w", err)
	}

	log.Info("VectorMigration reconciled after resource cleanup")
	return ctrl.Result{}, nil
}

func getStageVersionUsageName(stageVersionName string) string {
	return fmt.Sprintf("%s-%s", stageVersionName, "migration-usage")
}

// TODO same method as in stage controller, move to separate common library
func adaptVectorName(vector string) (string, error) {
	trimmedVector := strings.TrimSpace(strings.ToLower(vector))

	// TODO validate defined vector format
	if len(trimmedVector) < 4 {
		return "", fmt.Errorf("unable to parse vector: %s", vector)
	}

	// get index of separator
	separatorIdx := strings.LastIndex(trimmedVector, "//")

	if separatorIdx == -1 || separatorIdx == len(vector)-2 {
		return "", fmt.Errorf("unable to parse vector: %s", vector)
	}

	componentVersion := trimmedVector[separatorIdx+2:]
	adaptedVector := strings.ReplaceAll(componentVersion, "/", ".")
	adaptedVector = strings.ReplaceAll(adaptedVector, ":", "-")
	return adaptedVector, nil
}

func taskExecutionFailed(taskExecution landscape.TaskExecution) bool {
	return meta.IsStatusConditionTrue(taskExecution.Status.Conditions, landscape.TaskFailed)
}

func taskExecutionSucceeded(taskExecution landscape.TaskExecution) bool {
	return meta.IsStatusConditionTrue(taskExecution.Status.Conditions, landscape.TaskSucceeded)
}

func allTaskDependenciesSucceeded(task landscape.TaskManifest, successfulTasksByName map[string]bool) bool {
	if len(task.DependsOn) == 0 {
		return true
	}

	for _, dependency := range task.DependsOn {
		if !successfulTasksByName[dependency] {
			return false
		}
	}

	return true
}
