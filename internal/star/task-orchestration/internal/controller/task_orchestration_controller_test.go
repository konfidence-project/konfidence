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
	"time"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	testutil "github.com/konfidence-project/landscape-task-orchestration-controller/internal/utils"
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
		Vector001           = "https://registry.kdenv.lab/ocm/vector//landscape.konfidence.cloud/example/vector:0.0.1"
		VectorName001       = "landscape.konfidence.cloud.example.vector-0.0.1"
		Task0               = "task-0"
		Task1               = "task-1"
		Task2               = "task-2"
		Task3               = "task-3"
		Task4               = "task-4"
		Task5               = "task-5"
		Task6               = "task-6"
		SpecJson            = "{}"
		timeout             = time.Second * 10
		interval            = time.Millisecond * 250
	)

	BeforeEach(func() {
		testutil.CleanupArtifactDeployment(k8sClient, ArtifactDeployment1, Namespace)
		testutil.CleanupArtifactDeployment(k8sClient, ArtifactDeployment2, Namespace)
		testutil.CleanupVectorDeployment(k8sClient, VectorName001, Namespace)
		testutil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
		testutil.CleanupStageVersionUsage(k8sClient, StageVersionUsage, Namespace)
		testutil.CleanupVectorMigration(k8sClient, VectorMigration, Namespace)
		testutil.CleanupStage(k8sClient, StageDev, Namespace)
	})

	AfterEach(func() {
		testutil.CleanupArtifactDeployment(k8sClient, ArtifactDeployment1, Namespace)
		testutil.CleanupArtifactDeployment(k8sClient, ArtifactDeployment2, Namespace)
		testutil.CleanupVectorDeployment(k8sClient, VectorName001, Namespace)
		testutil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
		testutil.CleanupStageVersionUsage(k8sClient, StageVersionUsage, Namespace)
		testutil.CleanupVectorMigration(k8sClient, VectorMigration, Namespace)
		testutil.CleanupStage(k8sClient, StageDev, Namespace)
	})

	Context("When reconciling a vectorMigration", func() {
		It("should successfully reconcile the vectorMigration", func() {
			ctx := context.Background()

			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDev, Vector001)

			// check that the stage has been created and has valid properties
			stage := &landscape.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001, StageDev)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			// set stage as owner
			Expect(controllerutil.SetOwnerReference(stage, stageVersion, k8sClient.Scheme())).To(Succeed())
			testutil.UpdateStageVersion(ctx, k8sClient, stageVersion)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			artifactDeployment1Tasks := []landscape.TaskManifest{
				{
					Name: Task0,
					Type: "k8s",
					Spec: runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task3,
					Type:      "k8s",
					DependsOn: []string{Task0, Task2},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task5,
					Type:      "k8s",
					DependsOn: []string{Task3},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task6,
					Type:      "k8s",
					DependsOn: []string{Task4, Task5},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
			}

			// create ArtifactDeployments
			testutil.CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment1, Namespace, artifactDeployment1Tasks)

			artifactDeployment1 := &landscape.ArtifactDeployment{}
			artifactDeployment1LookupKey := types.NamespacedName{Name: ArtifactDeployment1, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment1LookupKey, artifactDeployment1)).To(Succeed())
				g.Expect(artifactDeployment1.Spec.TaskManifests).To(HaveLen(4))
			}, timeout, interval).Should(Succeed())

			artifactDeployment2Tasks := []landscape.TaskManifest{
				{
					Name: Task1,
					Type: "k8s",
					Spec: runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task2,
					Type:      "k8s",
					DependsOn: []string{Task0, Task1},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
				{
					Name:      Task4,
					Type:      "k8s",
					DependsOn: []string{Task2},
					Spec:      runtime.RawExtension{Raw: []byte(SpecJson)},
				},
			}

			testutil.CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment2, Namespace, artifactDeployment2Tasks)

			artifactDeployment2 := &landscape.ArtifactDeployment{}
			artifactDeployment2LookupKey := types.NamespacedName{Name: ArtifactDeployment2, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment2LookupKey, artifactDeployment2)).To(Succeed())
				g.Expect(artifactDeployment2.Spec.TaskManifests).To(HaveLen(3))
			}, timeout, interval).Should(Succeed())

			// create vectorDeployment
			testutil.CreateVectorDeployment(ctx, k8sClient, VectorName001, Namespace, Vector001, StageVersion)

			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			// check that the vectorDeployment has been created and has valid properties
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// update vector status and add artifactDeployment References
			artifactDeploymentRefs := make(map[string]landscape.LocalArtifactDeploymentReference)
			artifactDeploymentRefs[ArtifactDeployment1] = landscape.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment1,
			}
			artifactDeploymentRefs[ArtifactDeployment2] = landscape.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment2,
			}
			vectorDeployment.Status.ResultingArtifactDeployments = artifactDeploymentRefs
			testutil.UpdateVectorDeploymentStatus(ctx, k8sClient, vectorDeployment)

			// check that the status has been updated
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Status.ResultingArtifactDeployments).To(HaveLen(2))
			}, timeout, interval).Should(Succeed())

			// create the vectorMigration
			testutil.CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: VectorMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Spec.StageVersion).To(Equal(StageVersion))
				g.Expect(vectorMigration.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// check that a new stageVersionUsage has been created
			stageVersionUsage := &landscape.StageVersionUsage{}
			stageVersionUsageLookupKey := types.NamespacedName{Name: StageVersionUsage, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)).To(Succeed())
				g.Expect(stageVersionUsage.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(stageVersionUsage.Spec.StageVersionRef).ToNot(BeNil())
				g.Expect(stageVersionUsage.Spec.StageVersionRef.Name).To(Equal(StageVersion))
				g.Expect(testutil.HasOwnerReference(stageVersionUsage.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.VectorMigrationKind,
					Name: VectorMigration,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that the first two taskExecutions have been created
			taskExecution0 := &landscape.TaskExecution{}
			taskExecution0LookupKey := types.NamespacedName{Name: Task0, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, taskExecution0LookupKey, taskExecution0)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// check that the first two taskExecutions have been created
			taskExecution1 := &landscape.TaskExecution{}
			taskExecution1LookupKey := types.NamespacedName{Name: Task1, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, taskExecution1LookupKey, taskExecution1)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark task0 as successful
			testutil.SetTaskExecutionStatus(ctx, k8sClient, taskExecution0, landscape.TaskSucceeded)

			// check that taskExecution2 has not yet been created
			taskExecution2 := &landscape.TaskExecution{}
			taskExecution2LookupKey := types.NamespacedName{Name: Task2, Namespace: Namespace}
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, taskExecution2LookupKey, taskExecution2)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())

			// mark task1 as successful
			testutil.SetTaskExecutionStatus(ctx, k8sClient, taskExecution1, landscape.TaskSucceeded)

			// now taskExecution2 should have been created
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, taskExecution2LookupKey, taskExecution2)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark task2 as successful
			testutil.SetTaskExecutionStatus(ctx, k8sClient, taskExecution2, landscape.TaskSucceeded)

			// now taskExecution3 should have been created
			taskExecution3 := &landscape.TaskExecution{}
			taskExecution3LookupKey := types.NamespacedName{Name: Task3, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, taskExecution3LookupKey, taskExecution3)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// and also now taskExecution4 should have been created
			taskExecution4 := &landscape.TaskExecution{}
			taskExecution4LookupKey := types.NamespacedName{Name: Task4, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, taskExecution4LookupKey, taskExecution4)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark task4 as successful
			testutil.SetTaskExecutionStatus(ctx, k8sClient, taskExecution4, landscape.TaskSucceeded)

			// check that taskExecution5 has not yet been created
			taskExecution5 := &landscape.TaskExecution{}
			taskExecution5LookupKey := types.NamespacedName{Name: Task5, Namespace: Namespace}
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, taskExecution5LookupKey, taskExecution5)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())

			// check that taskExecution6 has not yet been created
			taskExecution6 := &landscape.TaskExecution{}
			taskExecution6LookupKey := types.NamespacedName{Name: Task6, Namespace: Namespace}
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, taskExecution6LookupKey, taskExecution6)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())

			// mark task3 as successful
			testutil.SetTaskExecutionStatus(ctx, k8sClient, taskExecution3, landscape.TaskSucceeded)

			// now taskExecution5 should have been created
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, taskExecution5LookupKey, taskExecution5)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark task5 as successful
			testutil.SetTaskExecutionStatus(ctx, k8sClient, taskExecution5, landscape.TaskSucceeded)

			// now taskExecution6 should have been created
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, taskExecution6LookupKey, taskExecution6)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark task6 as successful
			testutil.SetTaskExecutionStatus(ctx, k8sClient, taskExecution6, landscape.TaskSucceeded)

			// vectorMigration should be marked successful
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, landscape.VectorMigrationSucceeded)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that stageVersionUsage has been deleted
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, stageVersionUsageLookupKey, stageVersionUsage)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err).To(MatchError(errors.IsNotFound, "Should be a not found error"))
			}, timeout, interval).Should(Succeed())
		})

		It("should fail to reconcile the vectorMigration if at least one task fails", func() {
			ctx := context.Background()

			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDev, Vector001)

			// check that the stage has been created and has valid properties
			stage := &landscape.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001, StageDev)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			// set stage as owner
			Expect(controllerutil.SetOwnerReference(stage, stageVersion, k8sClient.Scheme())).To(Succeed())
			testutil.UpdateStageVersion(ctx, k8sClient, stageVersion)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			artifactDeploymentTasks := []landscape.TaskManifest{
				{
					Name: Task0,
					Type: "k8s",
					Spec: runtime.RawExtension{Raw: []byte(SpecJson)},
				},
			}

			// create ArtifactDeployment
			testutil.CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment1, Namespace, artifactDeploymentTasks)

			artifactDeployment1 := &landscape.ArtifactDeployment{}
			artifactDeployment1LookupKey := types.NamespacedName{Name: ArtifactDeployment1, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment1LookupKey, artifactDeployment1)).To(Succeed())
				g.Expect(artifactDeployment1.Spec.TaskManifests).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			// create vectorDeployment
			testutil.CreateVectorDeployment(ctx, k8sClient, VectorName001, Namespace, Vector001, StageVersion)

			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			// check that the vectorDeployment has been created and has valid properties
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// update vector status and add artifactDeployment References
			artifactDeploymentRefs := make(map[string]landscape.LocalArtifactDeploymentReference)
			artifactDeploymentRefs[ArtifactDeployment1] = landscape.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment1,
			}
			vectorDeployment.Status.ResultingArtifactDeployments = artifactDeploymentRefs
			testutil.UpdateVectorDeploymentStatus(ctx, k8sClient, vectorDeployment)

			// check that the status has been updated
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Status.ResultingArtifactDeployments).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			// create the vectorMigration
			testutil.CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &landscape.VectorMigration{}
			vectorMigrationLookupKey := types.NamespacedName{Name: VectorMigration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Spec.StageVersion).To(Equal(StageVersion))
				g.Expect(vectorMigration.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// check that the first taskExecution has been created
			taskExecution0 := &landscape.TaskExecution{}
			taskExecution0LookupKey := types.NamespacedName{Name: Task0, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, taskExecution0LookupKey, taskExecution0)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// mark task0 as failed
			testutil.SetTaskExecutionStatus(ctx, k8sClient, taskExecution0, landscape.TaskFailed)

			// vectorMigration should be marked as failed
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorMigrationLookupKey, vectorMigration)).To(Succeed())
				g.Expect(vectorMigration.Status.Conditions).To(HaveLen(1))
				g.Expect(meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, landscape.VectorMigrationFailed)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})

		It("should successfully reconcile the vectorMigration if no migration tasks exist", func() {
			ctx := context.Background()

			testutil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDev, Vector001)

			// check that the stage has been created and has valid properties
			stage := &landscape.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			testutil.CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001, StageDev)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			// set stage as owner
			Expect(controllerutil.SetOwnerReference(stage, stageVersion, k8sClient.Scheme())).To(Succeed())
			testutil.UpdateStageVersion(ctx, k8sClient, stageVersion)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testutil.HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: landscape.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// create ArtifactDeployment
			tasks := []landscape.TaskManifest{}
			testutil.CreateArtifactDeployment(ctx, k8sClient, ArtifactDeployment1, Namespace, tasks)

			artifactDeployment1 := &landscape.ArtifactDeployment{}
			artifactDeployment1LookupKey := types.NamespacedName{Name: ArtifactDeployment1, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, artifactDeployment1LookupKey, artifactDeployment1)).To(Succeed())
				g.Expect(artifactDeployment1.Spec.TaskManifests).To(BeEmpty())
			}, timeout, interval).Should(Succeed())

			// create vectorDeployment
			testutil.CreateVectorDeployment(ctx, k8sClient, VectorName001, Namespace, Vector001, StageVersion)

			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			// check that the vectorDeployment has been created and has valid properties
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// update vector status and add artifactDeployment References
			artifactDeploymentRefs := make(map[string]landscape.LocalArtifactDeploymentReference)
			artifactDeploymentRefs[ArtifactDeployment1] = landscape.LocalArtifactDeploymentReference{
				Name: ArtifactDeployment1,
			}
			vectorDeployment.Status.ResultingArtifactDeployments = artifactDeploymentRefs
			testutil.UpdateVectorDeploymentStatus(ctx, k8sClient, vectorDeployment)

			// check that the status has been updated
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Status.ResultingArtifactDeployments).To(HaveLen(1))
			}, timeout, interval).Should(Succeed())

			// create the vectorMigration
			testutil.CreateVectorMigration(ctx, k8sClient, VectorMigration, Namespace, StageVersion, Vector001)

			// check that the vectorMigration has been created and has valid properties
			vectorMigration := &landscape.VectorMigration{}
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
				g.Expect(meta.IsStatusConditionTrue(vectorMigration.Status.Conditions, landscape.VectorMigrationSucceeded)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
