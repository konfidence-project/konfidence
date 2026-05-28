package controller

import (
	"context"
	"fmt"
	"reflect"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/star/task-orchestration/internal/graph"
	pkgCtrl "github.com/konfidence-project/konfidence/pkg/controller"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	layerSucceeded                  = 0
	layerPartiallySucceeded         = 1
	layerPending                    = 2
	TaskOrchestrationControllerName = "task-orchestration-controller"
)

// TaskOrchestrationReconciler reconciles a VectorMigration object
type TaskOrchestrationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

type MapTasksResult struct {
	taskExecutionsByName           map[string]star.TaskExecution
	successfulTaskExecutionsByName map[string]bool
	taskFailed                     bool
}

// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectormigrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectormigrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversionusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=stageversionusages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=artifactdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=artifactdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=taskexecutions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=taskexecutions/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *TaskOrchestrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile started...")

	// get vectorMigration
	vectorMigration := &star.VectorMigration{}
	if err := r.Get(ctx, req.NamespacedName, vectorMigration); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, star.VectorMigrationSucceeded) {
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

func (r *TaskOrchestrationReconciler) reconcileVectorMigration(ctx context.Context, req ctrl.Request, vectorMigration *star.VectorMigration) error {
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

	if len(tasks) == 0 {
		log.Info("No migration tasks found")

		// mark vectorMigration as successful if no tasks exist
		meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{
			Type:               star.VectorMigrationSucceeded,
			Status:             metav1.ConditionTrue,
			Reason:             star.VectorMigrationSucceeded,
			Message:            fmt.Sprintf("Successfully reconciled VectorMigration %s", vectorMigration.Name),
			ObservedGeneration: vectorMigration.Generation,
			LastTransitionTime: metav1.Now(),
		})

		// delete stageVersionUsage
		if err := r.Delete(ctx, stageVersionUsage); err != nil {
			return fmt.Errorf("unable to delete stageVersionUsage for vectorMigration: %w", err)
		}
		r.Recorder.Eventf(vectorMigration, nil, corev1.EventTypeNormal,
			"StageVersionUsageDeleted", "StageVersionUsageDeleted",
			fmt.Sprintf("Deleted StageVersionUsage %s", stageVersionUsage.Name))

		log.Info("VectorMigration reconciled")
		return nil
	}

	// sort tasks
	_, layers, err := graph.SortTasks(tasks)
	if err != nil {
		return fmt.Errorf("unable to sort task dependency graph: %w", err)
	}

	// map taskExecutions by task name
	mappedTasks, err := r.mapTasks(ctx, req)
	if err != nil {
		return fmt.Errorf("mapping taskExecutions failed: %w", err)
	}

	if mappedTasks.taskFailed {
		// if at least one of the tasks failed mark vectorMigration as failed
		meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{
			Type:               star.VectorMigrationFailed,
			Status:             metav1.ConditionTrue,
			Reason:             star.VectorMigrationFailed,
			Message:            fmt.Sprintf("Reconciling VectorMigration %s failed", vectorMigration.Name),
			ObservedGeneration: vectorMigration.Generation,
			LastTransitionTime: metav1.Now(),
		})
		return fmt.Errorf("reconciling VectorMigration failed")
	}

	// process all layers
	layerStatus := make([]int, len(layers))
	for idx, layer := range layers {
		status, err := r.processTaskLayer(ctx, vectorMigration, layer, mappedTasks.taskExecutionsByName, mappedTasks.successfulTaskExecutionsByName)
		if err != nil {
			return fmt.Errorf("failed to process tasks: %w", err)
		}

		if status == layerPending {
			log.Info("Task layer still pending... retry later")
			// wait for taskExecution status change notifications
			return nil
		}

		layerStatus[idx] = status
	}

	// check that all layer tasks have actually succeeded
	for _, status := range layerStatus {
		if status != layerSucceeded {
			log.Info("Waiting for task layers to finish...")
			// wait for taskExecution status change notifications
			return nil
		}
	}

	log.Info("Finished processing tasks")

	// mark vectorMigration as successful
	meta.SetStatusCondition(&vectorMigration.Status.Conditions, metav1.Condition{
		Type:               star.VectorMigrationSucceeded,
		Status:             metav1.ConditionTrue,
		Reason:             star.VectorMigrationSucceeded,
		Message:            fmt.Sprintf("Successfully reconciled VectorMigration %s", vectorMigration.Name),
		ObservedGeneration: vectorMigration.Generation,
		LastTransitionTime: metav1.Now(),
	})

	log.Info("Cleaning up resources...")

	// delete taskExecutions
	if err := r.deleteTaskExecutions(ctx, maps.Values(mappedTasks.taskExecutionsByName), vectorMigration); err != nil {
		return fmt.Errorf("unable to delete taskExecutions: %w", err)
	}

	// delete stageVersionUsage
	if err := r.Delete(ctx, stageVersionUsage); err != nil {
		return fmt.Errorf("unable to delete stageVersionUsage for vectorMigration: %w", err)
	}
	r.Recorder.Eventf(vectorMigration, nil, corev1.EventTypeNormal,
		"StageVersionUsageDeleted", "StageVersionUsageDeleted",
		fmt.Sprintf("Deleted StageVersionUsage %s", stageVersionUsage.Name))
	log.Info("VectorMigration reconciled")
	return nil
}

func (r *TaskOrchestrationReconciler) getVectorDeployment(
	ctx context.Context, vectorMigration *star.VectorMigration,
) (*star.VectorDeployment, error) {
	labelMatcher := client.MatchingLabels{}
	labelMatcher[pkgCtrl.StageVersionNameLabel] = vectorMigration.Spec.StageVersion

	vectorDeployments := &star.VectorDeploymentList{}
	if err := r.List(ctx, vectorDeployments, client.InNamespace(vectorMigration.Namespace), labelMatcher); err != nil {
		return nil, fmt.Errorf("unable to list vectorDeployments: %w", err)
	}

	if len(vectorDeployments.Items) != 1 {
		return nil, fmt.Errorf("unable to find vectorDeployment for stage version %s", vectorMigration.Spec.StageVersion)
	}

	return &vectorDeployments.Items[0], nil
}

func (r *TaskOrchestrationReconciler) getArtifactDeployment(
	ctx context.Context, vectorMigration *star.VectorMigration, name string,
) (*star.ArtifactDeployment, error) {
	artifactDeployment := &star.ArtifactDeployment{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: vectorMigration.Namespace,
		Name:      name,
	}, artifactDeployment)

	return artifactDeployment, err
}

func (r *TaskOrchestrationReconciler) getTaskExecutions(ctx context.Context, req ctrl.Request) ([]star.TaskExecution, error) {
	// get all taskExecutions that are owned by this vectorMigration
	taskExecutions := &star.TaskExecutionList{}
	if err := r.List(ctx, taskExecutions, client.InNamespace(req.Namespace), client.MatchingFields{vectorMigrationOwnerKey: req.Name}); err != nil {
		return nil, fmt.Errorf("unable to list taskExecutions: %w", err)
	}

	return taskExecutions.Items, nil
}

func (r *TaskOrchestrationReconciler) deleteTaskExecutions(
	ctx context.Context, taskExecutions []star.TaskExecution, vectorMigration *star.VectorMigration,
) error {
	for _, taskExecution := range taskExecutions {
		if err := r.Delete(ctx, &taskExecution); err != nil {
			return fmt.Errorf("unable to delete taskExecution for vectorMigration: %w", err)
		}
		r.Recorder.Eventf(vectorMigration, nil, corev1.EventTypeNormal,
			"TaskExecutionDeleted", "TaskExecutionDeleted",
			fmt.Sprintf("Deleted TaskExecution %s", taskExecution.Name))
	}

	return nil
}

func (r *TaskOrchestrationReconciler) createOrGetStageVersionUsage(
	ctx context.Context, req ctrl.Request, vectorMigration *star.VectorMigration,
) (*star.StageVersionUsage, error) {
	log := logf.FromContext(ctx)
	stageVersionUsage := &star.StageVersionUsage{}
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
		msg := fmt.Sprintf("Created StageVersionUsage %s", stageVersionUsage.Name)
		r.Recorder.Eventf(vectorMigration, nil, corev1.EventTypeNormal, "StageVersionUsageCreated", "StageVersionUsageCreated", msg)
		log.V(1).Info(msg)
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

	return stageVersionUsage, nil
}

func (r *TaskOrchestrationReconciler) getArtifactDeploymentsAndTasks(
	ctx context.Context,
	vectorDeployment *star.VectorDeployment,
	vectorMigration *star.VectorMigration,
) ([]star.TaskManifest, error) {
	artifactDeployments := make(map[string]star.ArtifactDeployment)
	numberOfTasks := 0
	for _, deploymentReference := range vectorDeployment.Status.ResultingArtifactDeployments {
		// get artifactDeployment
		artifactDeployment, err := r.getArtifactDeployment(ctx, vectorMigration, deploymentReference.Name)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch artifactDeployment: %w", err)
		}

		artifactDeployments[deploymentReference.Name] = *artifactDeployment
		numberOfTasks = numberOfTasks + len(artifactDeployment.Spec.TaskManifests)
	}

	if numberOfTasks == 0 {
		return nil, nil
	}

	// get all task manifests from deployments
	tasks := make([]star.TaskManifest, 0, numberOfTasks)
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

	// map taskExecutions by task name
	mappedTasks := &MapTasksResult{}
	taskExecutionsByName := make(map[string]star.TaskExecution)
	successfulTaskExecutionsByName := make(map[string]bool)
	for _, taskExecution := range taskExecutions {
		if taskExecutionFailed(taskExecution) {
			log.Info("Task execution failed", "taskExecution", taskExecution)
			mappedTasks.taskFailed = true
			return mappedTasks, nil
		}

		taskExecutionsByName[taskExecution.Spec.Name] = taskExecution
		successfulTaskExecutionsByName[taskExecution.Spec.Name] = taskExecutionSucceeded(taskExecution)
	}

	mappedTasks.taskExecutionsByName = taskExecutionsByName
	mappedTasks.successfulTaskExecutionsByName = successfulTaskExecutionsByName
	return mappedTasks, nil
}

func (r *TaskOrchestrationReconciler) processTaskLayer(
	ctx context.Context,
	vectorMigration *star.VectorMigration,
	layer []star.TaskManifest,
	taskExecutionsByName map[string]star.TaskExecution,
	successfulTaskExecutionsByName map[string]bool,
) (int, error) {
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
				r.Recorder.Eventf(vectorMigration, nil, corev1.EventTypeNormal,
					"TaskExecutionCreated", "TaskExecutionCreated",
					fmt.Sprintf("Created TaskExecution %s", taskExecution.Name))
			}
		}

		if successfulTaskExecutionsByName[task.Name] {
			r.Recorder.Eventf(vectorMigration, nil, corev1.EventTypeNormal,
				"TaskExecutionSucceeded", "TaskExecutionSucceeded",
				fmt.Sprintf("TaskExecution %s succeeded", task.Name))
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
	apiGVStr                = star.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *TaskOrchestrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &star.TaskExecution{}, vectorMigrationOwnerKey, func(rawObj client.Object) []string {
		// grab the taskExecution object and extract the owner
		taskExecution := rawObj.(*star.TaskExecution)
		owner := metav1.GetControllerOf(taskExecution)
		if owner == nil {
			return nil
		}
		// make sure it is a stage...
		if owner.APIVersion != apiGVStr || owner.Kind != star.VectorMigrationKind {
			return nil
		}

		// and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return fmt.Errorf("unable to create index for owner reference of task execution: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&star.VectorMigration{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&star.TaskExecution{}).
		Named("taskOrchestration").
		Complete(r)
}

func (r *TaskOrchestrationReconciler) constructStageVersionUsage(vectorMigration *star.VectorMigration) (*star.StageVersionUsage, error) {
	stageVersionUsage := &star.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getStageVersionUsageName(vectorMigration.Spec.StageVersion),
			Namespace: vectorMigration.Namespace,
		},
		Spec: star.StageVersionUsageSpec{
			Reason: "VectorMigration",
			StageVersionRef: &star.StageVersionReference{
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

func (r *TaskOrchestrationReconciler) constructTaskExecution(
	vectorMigration *star.VectorMigration,
	taskManifest star.TaskManifest,
	namespace string,
) (*star.TaskExecution, error) {
	taskExecution := &star.TaskExecution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskManifest.Name + "-" + rand.String(8),
			Namespace: namespace,
		},
		Spec: star.TaskExecutionSpec(taskManifest),
	}

	// set vectorMigration as controller
	if err := ctrl.SetControllerReference(vectorMigration, taskExecution, r.Scheme); err != nil {
		return nil, fmt.Errorf("unable to set controller reference for taskExecution: %w", err)
	}
	return taskExecution, nil
}

func (r *TaskOrchestrationReconciler) cleanupVectorMigration(
	ctx context.Context, req ctrl.Request, vectorMigration *star.VectorMigration,
) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Cleanup vector migration")

	// get all taskExecutions for this vectorMigration
	taskExecutions, err := r.getTaskExecutions(ctx, req)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not get task executions: %w", err)
	}

	// and delete tasks if necessary
	if err := r.deleteTaskExecutions(ctx, taskExecutions, vectorMigration); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete task executions: %w", err)
	}

	// check if stageVersionUsage still exists and should be deleted
	stageVersionUsage := &star.StageVersionUsage{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      getStageVersionUsageName(vectorMigration.Spec.StageVersion),
		Namespace: req.Namespace,
	}, stageVersionUsage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.Delete(ctx, stageVersionUsage); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to delete stageVersionUsage for vectorMigration: %w", err)
	}
	r.Recorder.Eventf(vectorMigration, nil, corev1.EventTypeNormal,
		"StageVersionUsageDeleted", "StageVersionUsageDeleted",
		fmt.Sprintf("Deleted StageVersionUsage %s", stageVersionUsage.Name))

	log.Info("VectorMigration reconciled after resource cleanup")
	return ctrl.Result{}, nil
}

func getStageVersionUsageName(stageVersionName string) string {
	return fmt.Sprintf("%s-%s", stageVersionName, "migration")
}

func taskExecutionFailed(taskExecution star.TaskExecution) bool {
	return meta.IsStatusConditionTrue(taskExecution.Status.Conditions, star.TaskFailed)
}

func taskExecutionSucceeded(taskExecution star.TaskExecution) bool {
	return meta.IsStatusConditionTrue(taskExecution.Status.Conditions, star.TaskSucceeded)
}

func allTaskDependenciesSucceeded(task star.TaskManifest, successfulTasksByName map[string]bool) bool {
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
