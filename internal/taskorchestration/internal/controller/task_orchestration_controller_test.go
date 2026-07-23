package controller

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Task Orchestration Controller", func() {
	const (
		StageDev            = "stage-dev"
		StageVersion        = "stage-version-stage-dev"
		VectorMigration     = "stage-version-stage-dev-migration"
		StageVersionUsage   = "stage-version-stage-dev-migration"
		ArtifactDeployment1 = "artifact-deployment-1"
		ArtifactDeployment2 = "artifact-deployment-2"
		Namespace           = "default"
		Vector001           = "https://registry.kdenv.lab/ocm/vector//example.konfidence.cloud/example/vector:0.0.1"
		VectorName001       = "example.konfidence.cloud.example.vector-0.0.1"
		Task0               = "task-0"
		Task1               = "task-1"
		Task2               = "task-2"
		Task3               = "task-3"
		Task4               = "task-4"
		Task5               = "task-5"
		Task6               = "task-6"
		SpecJson            = "{}"
		TaskTypeK8s         = "k8s"
		timeout             = time.Second * 10
		interval            = time.Millisecond * 250
	)

	BeforeEach(func() {
		CleanupArtifactDeployment(k8sClient, ArtifactDeployment1, Namespace)
		CleanupArtifactDeployment(k8sClient, ArtifactDeployment2, Namespace)
		CleanupVectorDeployment(k8sClient, VectorName001, Namespace)
		CleanupStageVersion(k8sClient, StageVersion, Namespace)
		CleanupStageVersionUsage(k8sClient, StageVersionUsage, Namespace)
		CleanupVectorMigration(k8sClient, VectorMigration, Namespace)
		CleanupStage(k8sClient, StageDev, Namespace)
		CleanupTaskExecutions(k8sClient)
	})

	AfterEach(func() {
		CleanupArtifactDeployment(k8sClient, ArtifactDeployment1, Namespace)
		CleanupArtifactDeployment(k8sClient, ArtifactDeployment2, Namespace)
		CleanupVectorDeployment(k8sClient, VectorName001, Namespace)
		CleanupStageVersion(k8sClient, StageVersion, Namespace)
		CleanupStageVersionUsage(k8sClient, StageVersionUsage, Namespace)
		CleanupVectorMigration(k8sClient, VectorMigration, Namespace)
		CleanupStage(k8sClient, StageDev, Namespace)
		CleanupTaskExecutions(k8sClient)
	})

	Context("When reconciling a vectorMigration", func() {
		It("should successfully reconcile the vectorMigration", func() {
			ctx := context.Background()

			CreateStage(ctx, k8sClient, StageDev, Namespace, StageDev, Vector001)

			// check that the stage has been created and has valid properties
			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001, StageDev)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &konfidence.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			// set stage as owner
			Expect(controllerutil.SetOwnerReference(stage, stageVersion, k8sClient.Scheme())).To(Succeed())
			UpdateStageVersion(ctx, k8sClient, stageVersion)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: konfidence.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			artifactDeployment1Tasks := []konfidence.TaskManifest{
				{
					Name: Task0,
					Type: TaskTypeK8s,
					Spec: runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task3,
					Type:      TaskTypeK8s,
					DependsOn: []string{Task0, Task2},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task5,
					Type:      TaskTypeK8s,
					DependsOn: []string{Task3},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task6,
					Type:      TaskTypeK8s,
					DependsOn: []string{Task4, Task5},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
			}

			// create ArtifactDeployments
			CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment1, Namespace, artifactDeployment1Tasks)

			artifactDeployment1 := &konfidence.ArtifactDeployment{}
			artifactDeployment1LookupKey := types.NamespacedName{Name: ArtifactDeployment1, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment1LookupKey, artifactDeployment1)).To(Succeed())
				g.Expect(artifactDeployment1.Spec.TaskManifests).To(HaveLen(4))
			}, timeout, interval).Should(Succeed())

			artifactDeployment2Tasks := []konfidence.TaskManifest{
				{
					Name: Task1,
					Type: TaskTypeK8s,
					Spec: runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task2,
					Type:      TaskTypeK8s,
					DependsOn: []string{Task0, Task1},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task4,
					Type:      TaskTypeK8s,
					DependsOn: []string{Task2},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
			}

			CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment2, Namespace, artifactDeployment2Tasks)

			artifactDeployment2 := &konfidence.ArtifactDeployment{}
			artifactDeployment2LookupKey := types.NamespacedName{Name: ArtifactDeployment2, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment2LookupKey, artifactDeployment2)).To(Succeed())
				g.Expect(artifactDeployment2.Spec.TaskManifests).To(HaveLen(3))
			}, timeout, interval).Should(Succeed())

			// create vectorDeployment
			CreateVectorDeployment(ctx, k8sClient, VectorName001, Namespace, Vector001, StageVersion)

			vectorDeployment := &konfidence.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			// check that the vectorDeployment has been created and has valid properties
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// update vector status and add artifactDeployment References
			artifactDeploymentRefs := make(map[string]konfidence.LocalArtifactDeploymentReference)
			artifactDeploymentRefs[ArtifactDeployment1] = konfidence.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment1,
			}
			artifactDeploymentRefs[ArtifactDeployment2] = konfidence.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment2,
			}
			vectorDeployment.Status.ResultingArtifactDeployments = artifactDeploymentRefs
			UpdateVectorDeploymentStatus(ctx, k8sClient, vectorDeployment)

			// check that the status has been updated
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Status.ResultingArtifactDeployments).To(HaveLen(2))
			}, timeout, interval).Should(Succeed())

			// create the vectorMigration
			CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &konfidence.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: VectorMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Spec.StageVersion).To(Equal(StageVersion))
				g.Expect(vectorMigration.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// check that a new stageVersionUsage has been created
			stageVersionUsage := &konfidence.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(stageVersionUsage.Spec.StageVersionRef).ToNot(BeNil())
				g.Expect(stageVersionUsage.Spec.StageVersionRef.Name).To(Equal(StageVersion))
				g.Expect(HasOwnerReference(stageVersionUsage.GetOwnerReferences(), metav1.OwnerReference{
					Kind: konfidence.VectorMigrationKind,
					Name: VectorMigration,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the first two taskExecutions have been created
			taskExecutions := &konfidence.TaskExecutionList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(2))
			}, timeout, interval).Should(Succeed())

			taskExecution0 := GetTaskExecutionWithTaskName(Task0, taskExecutions.Items)
			taskExecution1 := GetTaskExecutionWithTaskName(Task1, taskExecutions.Items)
			Expect(taskExecution0).ToNot(BeNil())
			Expect(taskExecution1).ToNot(BeNil())

			// mark task0 as successful
			SetTaskExecutionStatus(ctx, k8sClient, taskExecution0, konfidence.TaskSucceeded)

			// check that taskExecution2 has not yet been created
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(2))
			}, timeout, interval).Should(Succeed())

			taskExecution2 := GetTaskExecutionWithTaskName(Task2, taskExecutions.Items)
			Expect(taskExecution2).To(BeNil())

			// mark task1 as successful
			SetTaskExecutionStatus(ctx, k8sClient, taskExecution1, konfidence.TaskSucceeded)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(3))
			}, timeout, interval).Should(Succeed())

			// now taskExecution2 should have been created
			taskExecution2 = GetTaskExecutionWithTaskName(Task2, taskExecutions.Items)
			Expect(taskExecution2).ToNot(BeNil())

			// mark task2 as successful
			SetTaskExecutionStatus(ctx, k8sClient, taskExecution2, konfidence.TaskSucceeded)

			// now taskExecution3 and taskExecution4 should have been created
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(5))
			}, timeout, interval).Should(Succeed())

			taskExecution3 := GetTaskExecutionWithTaskName(Task3, taskExecutions.Items)
			taskExecution4 := GetTaskExecutionWithTaskName(Task4, taskExecutions.Items)
			Expect(taskExecution3).ToNot(BeNil())
			Expect(taskExecution4).ToNot(BeNil())

			// mark task4 as successful
			SetTaskExecutionStatus(ctx, k8sClient, taskExecution4, konfidence.TaskSucceeded)

			// check that taskExecution5 and taskExecution6 have not yet been created
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(5))
			}, timeout, interval).Should(Succeed())

			taskExecution5 := GetTaskExecutionWithTaskName(Task5, taskExecutions.Items)
			taskExecution6 := GetTaskExecutionWithTaskName(Task6, taskExecutions.Items)
			Expect(taskExecution5).To(BeNil())
			Expect(taskExecution6).To(BeNil())

			// mark task3 as successful
			SetTaskExecutionStatus(ctx, k8sClient, taskExecution3, konfidence.TaskSucceeded)

			// now taskExecution5 should have been created
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(6))
			}, timeout, interval).Should(Succeed())

			taskExecution5 = GetTaskExecutionWithTaskName(Task5, taskExecutions.Items)
			Expect(taskExecution5).ToNot(BeNil())
			taskExecution6 = GetTaskExecutionWithTaskName(Task6, taskExecutions.Items)
			Expect(taskExecution6).To(BeNil())

			// mark task5 as successful
			SetTaskExecutionStatus(ctx, k8sClient, taskExecution5, konfidence.TaskSucceeded)

			// now taskExecution6 should have been created
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(7))
			}, timeout, interval).Should(Succeed())

			taskExecution6 = GetTaskExecutionWithTaskName(Task6, taskExecutions.Items)
			Expect(taskExecution6).ToNot(BeNil())

			// mark task6 as successful
			SetTaskExecutionStatus(ctx, k8sClient, taskExecution6, konfidence.TaskSucceeded)

			// vectorMigration should be marked successful
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, konfidence.VectorMigrationSucceeded)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that stageVersionUsage has been deleted
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())

			// check that all taskExecutions have been cleaned up
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})

		It("should handle re-reconciliation of an already succeeded vectorMigration idempotently", func() {
			ctx := context.Background()

			// Create the stageVersion referenced by the migration.
			CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001, StageDev)

			stageVersion := &konfidence.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// Create a single migration task behind one artifactDeployment.
			artifactDeploymentTasks := []konfidence.TaskManifest{
				{
					Name: Task0,
					Type: TaskTypeK8s,
					Spec: runtime.RawExtension{Raw: []byte(SpecJson)},
				},
			}
			CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment1, Namespace, artifactDeploymentTasks)

			artifactDeployment1 := &konfidence.ArtifactDeployment{}
			artifactDeployment1LookupKey := types.NamespacedName{Name: ArtifactDeployment1, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment1LookupKey, artifactDeployment1)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// Create the vectorDeployment and link it to the artifactDeployment.
			CreateVectorDeployment(ctx, k8sClient, VectorName001, Namespace, Vector001, StageVersion)

			vectorDeployment := &konfidence.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			artifactDeploymentRefs := make(map[string]konfidence.LocalArtifactDeploymentReference)
			artifactDeploymentRefs[ArtifactDeployment1] = konfidence.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment1,
			}
			vectorDeployment.Status.ResultingArtifactDeployments = artifactDeploymentRefs
			UpdateVectorDeploymentStatus(ctx, k8sClient, vectorDeployment)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Status.ResultingArtifactDeployments).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			// Start the migration.
			CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			vectorMigration := &konfidence.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: VectorMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// Let the single task run to completion.
			taskExecutions := &konfidence.TaskExecutionList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			taskExecution0 := GetTaskExecutionWithTaskName(Task0, taskExecutions.Items)
			Expect(taskExecution0).ToNot(BeNil())

			SetTaskExecutionStatus(ctx, k8sClient, taskExecution0, konfidence.TaskSucceeded)

			// Wait for cleanup of child resources created by the successful migration.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, konfidence.VectorMigrationSucceeded)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// Trigger another reconcile after the migration already succeeded.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(BeEmpty())
			}, timeout, interval).Should(Succeed())

			stageVersionUsage := &konfidence.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			if vectorMigration.Labels == nil {
				vectorMigration.Labels = make(map[string]string)
			}
			vectorMigration.Labels["re-reconcile-trigger"] = "true"
			Expect(k8sClient.Update(ctx, vectorMigration)).To(Succeed())

			// Re-reconciliation must not recreate work or lock the stageVersion again.
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, konfidence.VectorMigrationSucceeded)).To(BeTrue())
			}, time.Second*2, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(BeEmpty())
			}, time.Second*2, interval).Should(Succeed())

			Consistently(func(g Gomega) {
				err := k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, time.Second*2, interval).Should(Succeed())
		})

		It("should fail to reconcile the vectorMigration if at least one task fails", func() {
			ctx := context.Background()

			CreateStage(ctx, k8sClient, StageDev, Namespace, StageDev, Vector001)

			// check that the stage has been created and has valid properties
			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001, StageDev)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &konfidence.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			// set stage as owner
			Expect(controllerutil.SetOwnerReference(stage, stageVersion, k8sClient.Scheme())).To(Succeed())
			UpdateStageVersion(ctx, k8sClient, stageVersion)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: konfidence.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			artifactDeploymentTasks := []konfidence.TaskManifest{
				{
					Name: Task0,
					Type: TaskTypeK8s,
					Spec: runtime.RawExtension{Raw: []byte(SpecJson)},
				},
			}

			// create ArtifactDeployment
			CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment1, Namespace, artifactDeploymentTasks)

			artifactDeployment1 := &konfidence.ArtifactDeployment{}
			artifactDeployment1LookupKey := types.NamespacedName{Name: ArtifactDeployment1, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment1LookupKey, artifactDeployment1)).To(Succeed())
				g.Expect(artifactDeployment1.Spec.TaskManifests).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			// create vectorDeployment
			CreateVectorDeployment(ctx, k8sClient, VectorName001, Namespace, Vector001, StageVersion)

			vectorDeployment := &konfidence.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			// check that the vectorDeployment has been created and has valid properties
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// update vector status and add artifactDeployment References
			artifactDeploymentRefs := make(map[string]konfidence.LocalArtifactDeploymentReference)
			artifactDeploymentRefs[ArtifactDeployment1] = konfidence.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment1,
			}
			vectorDeployment.Status.ResultingArtifactDeployments = artifactDeploymentRefs
			UpdateVectorDeploymentStatus(ctx, k8sClient, vectorDeployment)

			// check that the status has been updated
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Status.ResultingArtifactDeployments).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			// create the vectorMigration
			CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &konfidence.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: VectorMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Spec.StageVersion).To(Equal(StageVersion))
				g.Expect(vectorMigration.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// check that the first taskExecution has been created
			taskExecutions := &konfidence.TaskExecutionList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, taskExecutions)).To(Succeed())
				g.Expect(taskExecutions.Items).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			taskExecution0 := GetTaskExecutionWithTaskName(Task0, taskExecutions.Items)
			Expect(taskExecution0).ToNot(BeNil())

			// mark task0 as failed
			SetTaskExecutionStatus(ctx, k8sClient, taskExecution0, konfidence.TaskFailed)

			// vectorMigration should be marked as failed
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, konfidence.VectorMigrationFailed)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})

		It("should successfully reconcile the vectorMigration if no migration tasks exist", func() {
			ctx := context.Background()

			CreateStage(ctx, k8sClient, StageDev, Namespace, StageDev, Vector001)

			// check that the stage has been created and has valid properties
			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001, StageDev)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &konfidence.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			// set stage as owner
			Expect(controllerutil.SetOwnerReference(stage, stageVersion, k8sClient.Scheme())).To(Succeed())
			UpdateStageVersion(ctx, k8sClient, stageVersion)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: konfidence.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// create ArtifactDeployment
			tasks := []konfidence.TaskManifest{}
			CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment1, Namespace, tasks)

			artifactDeployment1 := &konfidence.ArtifactDeployment{}
			artifactDeployment1LookupKey := types.NamespacedName{Name: ArtifactDeployment1, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment1LookupKey, artifactDeployment1)).To(Succeed())
				g.Expect(artifactDeployment1.Spec.TaskManifests).To(BeEmpty())
			}, timeout, interval).Should(Succeed())

			// create vectorDeployment
			CreateVectorDeployment(ctx, k8sClient, VectorName001, Namespace, Vector001, StageVersion)

			vectorDeployment := &konfidence.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			// check that the vectorDeployment has been created and has valid properties
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// update vector status and add artifactDeployment References
			artifactDeploymentRefs := make(map[string]konfidence.LocalArtifactDeploymentReference)
			artifactDeploymentRefs[ArtifactDeployment1] = konfidence.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment1,
			}
			vectorDeployment.Status.ResultingArtifactDeployments = artifactDeploymentRefs
			UpdateVectorDeploymentStatus(ctx, k8sClient, vectorDeployment)

			// check that the status has been updated
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Status.ResultingArtifactDeployments).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			// create the vectorMigration
			CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &konfidence.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: VectorMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Spec.StageVersion).To(Equal(StageVersion))
				g.Expect(vectorMigration.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// vectorMigration should be marked successful
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, konfidence.VectorMigrationSucceeded)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
